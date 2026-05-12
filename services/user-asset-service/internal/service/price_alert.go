package service

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	authclient "gtrade/services/user-asset-service/internal/client/auth"
	"gtrade/services/user-asset-service/internal/client/catalog"
	notificationclient "gtrade/services/user-asset-service/internal/client/notification"
	"gtrade/services/user-asset-service/internal/repository"
)

type priceAlertRepository interface {
	ListNotificationSubscriptions(ctx context.Context) ([]repository.NotificationSubscription, error)
	ListWatchlistNotificationStates(ctx context.Context, watchlistItemID int64) ([]repository.WatchlistNotificationState, error)
	UpsertWatchlistNotificationState(ctx context.Context, state repository.WatchlistNotificationState) error
	GetUserNotificationDispatchState(ctx context.Context, userID int64) (*repository.UserNotificationDispatchState, error)
	UpsertUserNotificationDispatchState(ctx context.Context, userID int64, processedOn time.Time, sentAt *time.Time) error
}

type catalogPriceClient interface {
	GetItem(ctx context.Context, id string) (*catalog.Item, error)
	GetPriceHistory(ctx context.Context, id, gameMode string, limit int) (*catalog.PriceHistoryResponse, error)
}

type authContactClient interface {
	GetUserContact(ctx context.Context, userID int64) (*authclient.UserContact, error)
	ListUserContacts(ctx context.Context, verifiedOnly bool) ([]authclient.UserContact, error)
}

type notificationEmailClient interface {
	SendEmail(ctx context.Context, input notificationclient.SendEmailRequest) error
}

type PriceAlertChange struct {
	ItemID         string
	ItemName       string
	ImageURL       string
	Game           string
	Source         string
	GameMode       string
	Currency       string
	CurrentValue   float64
	PreviousValue  float64
	CollectedOn    string
	CollectedAt    time.Time
	AbsoluteChange float64
}

type pendingUserNotification struct {
	mode         string
	userID       int64
	changes      []PriceAlertChange
	stateUpdates []repository.WatchlistNotificationState
}

type ManualDispatchResult struct {
	TargetUserID  int64
	Mode          string
	UsersChecked  int
	EmailsSent    int
	ChangesFound  int
	UsersWithDiff int
	UsersSkipped  int
}

type AdminMessageResult struct {
	TargetUserID int64
	UsersChecked int
	EmailsSent   int
}

type PriceAlertService struct {
	repo         priceAlertRepository
	catalog      catalogPriceClient
	auth         authContactClient
	notification notificationEmailClient
	logger       zerolog.Logger
}

func NewPriceAlertService(logger zerolog.Logger, repo priceAlertRepository, catalogClient catalogPriceClient, authClient authContactClient, notificationClient notificationEmailClient) *PriceAlertService {
	return &PriceAlertService{
		repo:         repo,
		catalog:      catalogClient,
		auth:         authClient,
		notification: notificationClient,
		logger:       logger.With().Str("component", "price_alert").Logger(),
	}
}

func (s *PriceAlertService) RunCycle(ctx context.Context, now time.Time) error {
	if s == nil || s.repo == nil || s.catalog == nil || s.auth == nil || s.notification == nil {
		return nil
	}

	subscriptions, err := s.repo.ListNotificationSubscriptions(ctx)
	if err != nil {
		return fmt.Errorf("list notification subscriptions: %w", err)
	}

	immediate := map[int64]*pendingUserNotification{}
	digest := map[int64]*pendingUserNotification{}
	dueDigestUsers := map[int64]bool{}

	for _, sub := range subscriptions {
		if !sub.NotificationsEnabled {
			s.logSkip(sub, "notifications_disabled", nil)
			continue
		}

		item, err := s.catalog.GetItem(ctx, sub.ItemID)
		if err != nil || item == nil || !item.IsActive {
			reason := "item_inactive"
			switch {
			case err != nil:
				reason = "catalog_error"
			case item == nil:
				reason = "item_not_found"
			}
			s.logSkip(sub, reason, err)
			continue
		}

		historyResp, err := s.catalog.GetPriceHistory(ctx, sub.ItemID, "", 10)
		if err != nil || historyResp == nil {
			reason := "price_history_nil"
			if err != nil {
				reason = "price_history_error"
			}
			s.logSkip(sub, reason, err)
			continue
		}

		states, err := s.repo.ListWatchlistNotificationStates(ctx, sub.WatchlistID)
		if err != nil {
			return fmt.Errorf("list notification states: %w", err)
		}

		changes, initStates, sendStates := detectPriceAlertChanges(sub, item, historyResp.History, states, now)
		for _, state := range initStates {
			if err := s.repo.UpsertWatchlistNotificationState(ctx, state); err != nil {
				return fmt.Errorf("init notification state: %w", err)
			}
		}

		if sub.NotificationMode == "immediate" {
			if len(changes) == 0 {
				continue
			}
			pending := immediate[sub.UserID]
			if pending == nil {
				pending = &pendingUserNotification{mode: "immediate", userID: sub.UserID}
				immediate[sub.UserID] = pending
			}
			pending.changes = append(pending.changes, changes...)
			pending.stateUpdates = append(pending.stateUpdates, sendStates...)
			continue
		}

		due, err := s.shouldProcessDigest(ctx, sub.UserID, sub.NotificationTime, now)
		if err != nil {
			return err
		}
		if !due {
			continue
		}
		dueDigestUsers[sub.UserID] = true

		if len(changes) == 0 {
			continue
		}
		pending := digest[sub.UserID]
		if pending == nil {
			pending = &pendingUserNotification{mode: "daily_digest", userID: sub.UserID}
			digest[sub.UserID] = pending
		}
		pending.changes = append(pending.changes, changes...)
		pending.stateUpdates = append(pending.stateUpdates, sendStates...)
	}

	for _, pending := range immediate {
		if _, err := s.dispatchUserNotification(ctx, pending, now, false); err != nil {
			s.logger.Error().
				Int64("user_id", pending.userID).
				Int("changes", len(pending.changes)).
				Err(err).
				Msg("immediate dispatch failed, continuing cycle")
		}
	}

	for userID := range dueDigestUsers {
		pending := digest[userID]
		if pending != nil {
			if _, err := s.dispatchUserNotification(ctx, pending, now, true); err != nil {
				s.logger.Error().
					Int64("user_id", userID).
					Int("changes", len(pending.changes)).
					Err(err).
					Msg("digest dispatch failed, continuing cycle")
			}
			continue
		}
		if err := s.repo.UpsertUserNotificationDispatchState(ctx, userID, dayStartUTC(now), nil); err != nil {
			return fmt.Errorf("mark empty digest processed: %w", err)
		}
	}

	return nil
}

func (s *PriceAlertService) SendManualPriceAlerts(ctx context.Context, userID int64, forceSend bool, now time.Time) (*ManualDispatchResult, error) {
	if s == nil || s.repo == nil || s.catalog == nil || s.auth == nil || s.notification == nil {
		return &ManualDispatchResult{TargetUserID: userID}, nil
	}
	if forceSend {
		return s.sendForcedSnapshot(ctx, userID, now)
	}

	subscriptions, err := s.repo.ListNotificationSubscriptions(ctx)
	if err != nil {
		return nil, fmt.Errorf("list notification subscriptions: %w", err)
	}

	pendingByUser := map[int64]*pendingUserNotification{}
	checkedUsers := map[int64]bool{}

	for _, sub := range subscriptions {
		if userID > 0 && sub.UserID != userID {
			continue
		}
		if !sub.NotificationsEnabled {
			s.logSkip(sub, "notifications_disabled", nil)
			continue
		}

		checkedUsers[sub.UserID] = true

		item, err := s.catalog.GetItem(ctx, sub.ItemID)
		if err != nil || item == nil || !item.IsActive {
			reason := "item_inactive"
			switch {
			case err != nil:
				reason = "catalog_error"
			case item == nil:
				reason = "item_not_found"
			}
			s.logSkip(sub, reason, err)
			continue
		}

		historyResp, err := s.catalog.GetPriceHistory(ctx, sub.ItemID, "", 10)
		if err != nil || historyResp == nil {
			reason := "price_history_nil"
			if err != nil {
				reason = "price_history_error"
			}
			s.logSkip(sub, reason, err)
			continue
		}

		states, err := s.repo.ListWatchlistNotificationStates(ctx, sub.WatchlistID)
		if err != nil {
			return nil, fmt.Errorf("list notification states: %w", err)
		}

		changes, initStates, sendStates := detectPriceAlertChanges(sub, item, historyResp.History, states, now)
		for _, state := range initStates {
			if err := s.repo.UpsertWatchlistNotificationState(ctx, state); err != nil {
				return nil, fmt.Errorf("init notification state: %w", err)
			}
		}

		if len(changes) == 0 {
			continue
		}

		pending := pendingByUser[sub.UserID]
		if pending == nil {
			pending = &pendingUserNotification{mode: "immediate", userID: sub.UserID}
			pendingByUser[sub.UserID] = pending
		}
		pending.changes = append(pending.changes, changes...)
		pending.stateUpdates = append(pending.stateUpdates, sendStates...)
	}

	result := &ManualDispatchResult{
		TargetUserID: userID,
		Mode:         "diff",
		UsersChecked: len(checkedUsers),
	}

	for _, pending := range pendingByUser {
		result.ChangesFound += len(pending.changes)
		if len(pending.changes) > 0 {
			result.UsersWithDiff++
		}
		sent, err := s.dispatchUserNotification(ctx, pending, now, false)
		if err != nil {
			s.logger.Error().
				Int64("user_id", pending.userID).
				Int("changes", len(pending.changes)).
				Err(err).
				Msg("manual diff dispatch failed, continuing with remaining users")
			result.UsersSkipped++
			continue
		}
		if sent {
			result.EmailsSent++
		} else {
			result.UsersSkipped++
		}
	}

	return result, nil
}

// sendForcedSnapshot ignores detectPriceAlertChanges and sends every
// subscribed user a one-shot snapshot of their current watchlist values. The
// notification state is NOT updated, so the next regular cycle still behaves
// as if no manual send happened.
func (s *PriceAlertService) sendForcedSnapshot(ctx context.Context, userID int64, now time.Time) (*ManualDispatchResult, error) {
	subscriptions, err := s.repo.ListNotificationSubscriptions(ctx)
	if err != nil {
		return nil, fmt.Errorf("list notification subscriptions: %w", err)
	}

	s.logger.Info().
		Int64("target_user_id", userID).
		Int("subscriptions_total", len(subscriptions)).
		Msg("force_send: loaded subscriptions")

	pendingByUser := map[int64]*pendingUserNotification{}
	checkedUsers := map[int64]bool{}
	skippedByUser := map[int64]int{}

	for _, sub := range subscriptions {
		if userID > 0 && sub.UserID != userID {
			continue
		}
		if !sub.NotificationsEnabled {
			s.logger.Info().
				Int64("user_id", sub.UserID).
				Str("item_id", sub.ItemID).
				Str("skip_reason", "notifications_disabled").
				Msg("force_send: subscription skipped")
			skippedByUser[sub.UserID]++
			continue
		}
		checkedUsers[sub.UserID] = true

		item, err := s.catalog.GetItem(ctx, sub.ItemID)
		if err != nil || item == nil || !item.IsActive {
			reason := "item_inactive"
			switch {
			case err != nil:
				reason = "catalog_error"
			case item == nil:
				reason = "item_not_found"
			}
			s.logger.Info().
				Int64("user_id", sub.UserID).
				Str("item_id", sub.ItemID).
				Str("skip_reason", reason).
				Err(err).
				Msg("force_send: subscription skipped")
			skippedByUser[sub.UserID]++
			continue
		}

		historyResp, err := s.catalog.GetPriceHistory(ctx, sub.ItemID, "", 1)
		if err != nil || historyResp == nil || len(historyResp.History) == 0 {
			if err != nil {
				s.logger.Info().
					Int64("user_id", sub.UserID).
					Str("item_id", sub.ItemID).
					Str("skip_reason", "price_history_error").
					Err(err).
					Msg("force_send: subscription skipped")
				skippedByUser[sub.UserID]++
				continue
			}

			s.logger.Info().
				Int64("user_id", sub.UserID).
				Str("item_id", sub.ItemID).
				Msg("force_send: including subscription without price history")

			change := PriceAlertChange{
				ItemID:      item.ID,
				ItemName:    item.Name,
				ImageURL:    item.ImageURL,
				Game:        item.Game,
				Source:      item.Source,
				Currency:    sub.Currency,
				CollectedOn: "price unavailable",
			}

			pending := pendingByUser[sub.UserID]
			if pending == nil {
				pending = &pendingUserNotification{mode: "snapshot", userID: sub.UserID}
				pendingByUser[sub.UserID] = pending
			}
			pending.changes = append(pending.changes, change)
			continue
		}

		latest := historyResp.History[0]
		change := PriceAlertChange{
			ItemID:         item.ID,
			ItemName:       item.Name,
			ImageURL:       item.ImageURL,
			Game:           item.Game,
			Source:         latest.Source,
			GameMode:       latest.GameMode,
			Currency:       latest.Currency,
			CurrentValue:   latest.Value,
			PreviousValue:  latest.Value,
			CollectedOn:    latest.CollectedOn,
			CollectedAt:    latest.CollectedAt,
			AbsoluteChange: 0,
		}

		pending := pendingByUser[sub.UserID]
		if pending == nil {
			pending = &pendingUserNotification{mode: "snapshot", userID: sub.UserID}
			pendingByUser[sub.UserID] = pending
		}
		pending.changes = append(pending.changes, change)
	}

	result := &ManualDispatchResult{
		TargetUserID: userID,
		Mode:         "snapshot",
		UsersChecked: len(checkedUsers),
	}

	for uid, skipped := range skippedByUser {
		if _, hasPending := pendingByUser[uid]; !hasPending {
			s.logger.Warn().
				Int64("user_id", uid).
				Int("skipped_subscriptions", skipped).
				Msg("force_send: user has no deliverable items, no email will be sent")
		}
	}

	for _, pending := range pendingByUser {
		result.ChangesFound += len(pending.changes)
		if len(pending.changes) > 0 {
			result.UsersWithDiff++
		}
		s.logger.Info().
			Int64("user_id", pending.userID).
			Int("changes", len(pending.changes)).
			Msg("force_send: dispatching email")
		sent, err := s.dispatchUserNotification(ctx, pending, now, false)
		if err != nil {
			s.logger.Error().
				Int64("user_id", pending.userID).
				Int("changes", len(pending.changes)).
				Err(err).
				Msg("force_send: dispatch failed, continuing with remaining users")
			result.UsersSkipped++
			continue
		}
		if sent {
			result.EmailsSent++
		} else {
			result.UsersSkipped++
		}
	}

	s.logger.Info().
		Int64("target_user_id", userID).
		Int("users_checked", result.UsersChecked).
		Int("emails_sent", result.EmailsSent).
		Int("users_skipped", result.UsersSkipped).
		Int("changes_found", result.ChangesFound).
		Msg("force_send: dispatch summary")

	return result, nil
}

func (s *PriceAlertService) SendAdminMessage(ctx context.Context, userID int64, subject, htmlBody, textBody string) (*AdminMessageResult, error) {
	subject = strings.TrimSpace(subject)
	htmlBody = strings.TrimSpace(htmlBody)
	textBody = strings.TrimSpace(textBody)
	if subject == "" {
		return nil, fmt.Errorf("subject is required")
	}
	if htmlBody == "" && textBody == "" {
		return nil, fmt.Errorf("html_body or text_body is required")
	}

	result := &AdminMessageResult{TargetUserID: userID}

	if userID > 0 {
		contact, err := s.auth.GetUserContact(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("get user contact: %w", err)
		}
		result.UsersChecked = 1
		if contact == nil || strings.TrimSpace(contact.Email) == "" || !contact.EmailVerified {
			return result, nil
		}
		if err := s.notification.SendEmail(ctx, notificationclient.SendEmailRequest{
			To:       contact.Email,
			Subject:  subject,
			HTMLBody: htmlBody,
			TextBody: textBody,
		}); err != nil {
			return nil, fmt.Errorf("send admin message: %w", err)
		}
		result.EmailsSent = 1
		return result, nil
	}

	contacts, err := s.auth.ListUserContacts(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("list user contacts: %w", err)
	}
	result.UsersChecked = len(contacts)
	for _, contact := range contacts {
		if strings.TrimSpace(contact.Email) == "" {
			continue
		}
		if err := s.notification.SendEmail(ctx, notificationclient.SendEmailRequest{
			To:       contact.Email,
			Subject:  subject,
			HTMLBody: htmlBody,
			TextBody: textBody,
		}); err != nil {
			return nil, fmt.Errorf("send admin message: %w", err)
		}
		result.EmailsSent++
	}
	return result, nil
}

func (s *PriceAlertService) shouldProcessDigest(ctx context.Context, userID int64, notificationTime string, now time.Time) (bool, error) {
	scheduledAt, err := parseTodayScheduleUTC(notificationTime, now)
	if err != nil {
		return false, nil
	}
	if now.UTC().Before(scheduledAt) {
		return false, nil
	}

	state, err := s.repo.GetUserNotificationDispatchState(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("get dispatch state: %w", err)
	}
	if state != nil && state.LastDigestProcessedOn != nil && sameDayUTC(*state.LastDigestProcessedOn, now) {
		return false, nil
	}
	return true, nil
}

func (s *PriceAlertService) dispatchUserNotification(ctx context.Context, pending *pendingUserNotification, now time.Time, markDigest bool) (bool, error) {
	if pending == nil {
		return false, nil
	}

	sort.Slice(pending.changes, func(i, j int) bool {
		if pending.changes[i].CollectedAt.Equal(pending.changes[j].CollectedAt) {
			return pending.changes[i].ItemName < pending.changes[j].ItemName
		}
		return pending.changes[i].CollectedAt.After(pending.changes[j].CollectedAt)
	})

	contact, err := s.auth.GetUserContact(ctx, pending.userID)
	if err != nil || contact == nil || strings.TrimSpace(contact.Email) == "" || !contact.EmailVerified {
		reason := "contact_unknown"
		switch {
		case err != nil:
			reason = "auth_error"
		case contact == nil:
			reason = "contact_nil"
		case strings.TrimSpace(contact.Email) == "":
			reason = "email_empty"
		case !contact.EmailVerified:
			reason = "email_not_verified"
		}
		s.logger.Info().
			Int64("user_id", pending.userID).
			Str("mode", pending.mode).
			Int("changes", len(pending.changes)).
			Str("skip_reason", reason).
			Err(err).
			Msg("price alert dispatch skipped")

		for _, state := range pending.stateUpdates {
			if upsertErr := s.repo.UpsertWatchlistNotificationState(ctx, withoutSentAt(state)); upsertErr != nil {
				return false, fmt.Errorf("advance state without email: %w", upsertErr)
			}
		}
		if markDigest {
			if upsertErr := s.repo.UpsertUserNotificationDispatchState(ctx, pending.userID, dayStartUTC(now), nil); upsertErr != nil {
				return false, fmt.Errorf("mark digest processed without email: %w", upsertErr)
			}
		}
		return false, nil
	}

	sent := false
	if len(pending.changes) > 0 {
		htmlBody, textBody, subject, err := renderPriceAlertEmail(pending.mode, pending.changes, now)
		if err != nil {
			return false, fmt.Errorf("render price alert email: %w", err)
		}
		if err := s.notification.SendEmail(ctx, notificationclient.SendEmailRequest{
			To:       contact.Email,
			Subject:  subject,
			HTMLBody: htmlBody,
			TextBody: textBody,
		}); err != nil {
			return false, fmt.Errorf("send price alert email: %w", err)
		}
		sent = true
	}

	sentAt := now.UTC()
	for _, state := range pending.stateUpdates {
		state.LastNotificationSentAt = &sentAt
		if err := s.repo.UpsertWatchlistNotificationState(ctx, state); err != nil {
			return false, fmt.Errorf("upsert notification state after send: %w", err)
		}
	}
	if markDigest {
		if err := s.repo.UpsertUserNotificationDispatchState(ctx, pending.userID, dayStartUTC(now), &sentAt); err != nil {
			return false, fmt.Errorf("upsert digest dispatch state: %w", err)
		}
	}
	return sent, nil
}

func detectPriceAlertChanges(
	sub repository.NotificationSubscription,
	item *catalog.Item,
	history []catalog.PriceHistoryEntry,
	existingStates []repository.WatchlistNotificationState,
	now time.Time,
) ([]PriceAlertChange, []repository.WatchlistNotificationState, []repository.WatchlistNotificationState) {
	stateByKey := make(map[string]repository.WatchlistNotificationState, len(existingStates))
	for _, state := range existingStates {
		stateByKey[stateKey(state.Source, state.GameMode)] = state
	}

	grouped := groupPriceHistory(history)
	var changes []PriceAlertChange
	var initStates []repository.WatchlistNotificationState
	var sendStates []repository.WatchlistNotificationState

	for _, group := range grouped {
		if len(group) == 0 {
			continue
		}

		latest := group[0]
		key := stateKey(latest.Source, latest.GameMode)
		currentState, ok := stateByKey[key]
		latestDay, err := time.Parse("2006-01-02", latest.CollectedOn)
		if err != nil {
			continue
		}
		stateUpdate := repository.WatchlistNotificationState{
			WatchlistItemID:         sub.WatchlistID,
			Source:                  latest.Source,
			GameMode:                latest.GameMode,
			LastNotifiedCollectedOn: timePtr(dayStartUTC(latestDay)),
			LastNotifiedValue:       float64Ptr(latest.Value),
			LastNotificationSentAt:  timePtr(now.UTC()),
		}

		if !ok || currentState.LastNotifiedCollectedOn == nil {
			stateUpdate.LastNotificationSentAt = nil
			initStates = append(initStates, stateUpdate)
			continue
		}

		shouldSend := latestDay.After(dayStartUTC(*currentState.LastNotifiedCollectedOn))
		if !shouldSend && currentState.LastNotifiedValue != nil && latestDay.Equal(dayStartUTC(*currentState.LastNotifiedCollectedOn)) {
			shouldSend = latest.Value != *currentState.LastNotifiedValue
		}
		if !shouldSend {
			continue
		}

		previousValue := latest.Value
		if latestDay.Equal(dayStartUTC(*currentState.LastNotifiedCollectedOn)) && currentState.LastNotifiedValue != nil {
			previousValue = *currentState.LastNotifiedValue
		} else if len(group) > 1 {
			previousValue = group[1].Value
		} else if currentState.LastNotifiedValue != nil {
			previousValue = *currentState.LastNotifiedValue
		}

		changes = append(changes, PriceAlertChange{
			ItemID:         item.ID,
			ItemName:       item.Name,
			ImageURL:       item.ImageURL,
			Game:           item.Game,
			Source:         latest.Source,
			GameMode:       latest.GameMode,
			Currency:       latest.Currency,
			CurrentValue:   latest.Value,
			PreviousValue:  previousValue,
			CollectedOn:    latest.CollectedOn,
			CollectedAt:    latest.CollectedAt,
			AbsoluteChange: latest.Value - previousValue,
		})
		sendStates = append(sendStates, stateUpdate)
	}

	return changes, initStates, sendStates
}

func groupPriceHistory(history []catalog.PriceHistoryEntry) [][]catalog.PriceHistoryEntry {
	groups := map[string][]catalog.PriceHistoryEntry{}
	keys := make([]string, 0)
	for _, entry := range history {
		key := stateKey(entry.Source, entry.GameMode)
		if _, ok := groups[key]; !ok {
			keys = append(keys, key)
		}
		groups[key] = append(groups[key], entry)
	}

	sort.Strings(keys)
	out := make([][]catalog.PriceHistoryEntry, 0, len(keys))
	for _, key := range keys {
		entries := groups[key]
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].CollectedOn == entries[j].CollectedOn {
				return entries[i].CollectedAt.After(entries[j].CollectedAt)
			}
			return entries[i].CollectedOn > entries[j].CollectedOn
		})
		out = append(out, entries)
	}
	return out
}

func renderPriceAlertEmail(mode string, changes []PriceAlertChange, now time.Time) (string, string, string, error) {
	if mode == "snapshot" {
		return renderSnapshotEmail(changes, now)
	}

	subject := "Price changes in your GTrade watchlist"
	title := "Immediate watchlist update"
	if mode == "daily_digest" {
		subject = "Your daily GTrade price digest"
		title = "Daily price digest"
	}

	view := struct {
		Title   string
		Date    string
		Changes []map[string]string
	}{
		Title: title,
		Date:  now.UTC().Format("2006-01-02 15:04 UTC"),
	}

	var text strings.Builder
	text.WriteString(title + "\n")
	text.WriteString("Generated at: " + view.Date + "\n\n")

	for _, change := range changes {
		modeLabel := change.GameMode
		if modeLabel == "" {
			modeLabel = "default"
		}
		view.Changes = append(view.Changes, map[string]string{
			"image_url":      change.ImageURL,
			"name":           change.ItemName,
			"game":           strings.ToUpper(change.Game),
			"source":         change.Source,
			"game_mode":      modeLabel,
			"current_value":  formatPrice(change.CurrentValue, change.Currency),
			"previous_value": formatPrice(change.PreviousValue, change.Currency),
			"delta":          formatDelta(change.AbsoluteChange, change.Currency),
			"collected_on":   change.CollectedOn,
		})

		text.WriteString("- " + change.ItemName + " [" + strings.ToUpper(change.Game) + "]")
		if change.GameMode != "" {
			text.WriteString(" (" + change.GameMode + ")")
		}
		text.WriteString(": " + formatPrice(change.PreviousValue, change.Currency) + " -> " + formatPrice(change.CurrentValue, change.Currency) + " (" + formatDelta(change.AbsoluteChange, change.Currency) + ")\n")
	}

	tpl := template.Must(template.New("price-alert-email").Parse(priceAlertEmailTemplate))
	var html bytes.Buffer
	if err := tpl.Execute(&html, view); err != nil {
		return "", "", "", err
	}

	return html.String(), text.String(), subject, nil
}

func renderSnapshotEmail(changes []PriceAlertChange, now time.Time) (string, string, string, error) {
	subject := "Your GTrade watchlist snapshot"
	title := "Watchlist snapshot"

	view := struct {
		Title string
		Date  string
		Items []map[string]string
	}{
		Title: title,
		Date:  now.UTC().Format("2006-01-02 15:04 UTC"),
	}

	var text strings.Builder
	text.WriteString(title + "\n")
	text.WriteString("Generated at: " + view.Date + "\n\n")

	for _, change := range changes {
		modeLabel := change.GameMode
		if modeLabel == "" {
			modeLabel = "default"
		}
		view.Items = append(view.Items, map[string]string{
			"image_url":     change.ImageURL,
			"name":          change.ItemName,
			"game":          strings.ToUpper(change.Game),
			"source":        change.Source,
			"game_mode":     modeLabel,
			"current_value": formatSnapshotPrice(change),
			"collected_on":  formatSnapshotCollectedOn(change),
		})

		text.WriteString("- " + change.ItemName + " [" + strings.ToUpper(change.Game) + "]")
		if change.GameMode != "" {
			text.WriteString(" (" + change.GameMode + ")")
		}
		text.WriteString(": " + formatSnapshotPrice(change) + " on " + formatSnapshotCollectedOn(change) + "\n")
	}

	tpl := template.Must(template.New("watchlist-snapshot-email").Parse(watchlistSnapshotEmailTemplate))
	var html bytes.Buffer
	if err := tpl.Execute(&html, view); err != nil {
		return "", "", "", err
	}

	return html.String(), text.String(), subject, nil
}

func parseTodayScheduleUTC(value string, now time.Time) (time.Time, error) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid time")
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return time.Time{}, err
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return time.Time{}, err
	}
	current := now.UTC()
	return time.Date(current.Year(), current.Month(), current.Day(), hour, minute, 0, 0, time.UTC), nil
}

func dayStartUTC(t time.Time) time.Time {
	utc := t.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

func sameDayUTC(a, b time.Time) bool {
	return dayStartUTC(a).Equal(dayStartUTC(b))
}

func stateKey(source, gameMode string) string {
	return source + "|" + gameMode
}

func withoutSentAt(state repository.WatchlistNotificationState) repository.WatchlistNotificationState {
	state.LastNotificationSentAt = nil
	return state
}

func (s *PriceAlertService) logSkip(sub repository.NotificationSubscription, reason string, err error) {
	s.logger.Info().
		Int64("user_id", sub.UserID).
		Int64("watchlist_id", sub.WatchlistID).
		Str("item_id", sub.ItemID).
		Str("skip_reason", reason).
		Err(err).
		Msg("price alert subscription skipped")
}

func float64Ptr(v float64) *float64 {
	return &v
}

func timePtr(v time.Time) *time.Time {
	return &v
}

func formatPrice(value float64, currency string) string {
	return fmt.Sprintf("%.2f %s", value, strings.ToUpper(strings.TrimSpace(currency)))
}

func formatDelta(value float64, currency string) string {
	prefix := ""
	if value > 0 {
		prefix = "+"
	}
	return fmt.Sprintf("%s%.2f %s", prefix, value, strings.ToUpper(strings.TrimSpace(currency)))
}

func formatSnapshotPrice(change PriceAlertChange) string {
	if strings.TrimSpace(change.CollectedOn) == "" || strings.EqualFold(strings.TrimSpace(change.CollectedOn), "price unavailable") {
		return "price unavailable"
	}
	return formatPrice(change.CurrentValue, change.Currency)
}

func formatSnapshotCollectedOn(change PriceAlertChange) string {
	if strings.TrimSpace(change.CollectedOn) == "" {
		return "price unavailable"
	}
	return change.CollectedOn
}

const priceAlertEmailTemplate = `
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{ .Title }}</title>
</head>
<body style="margin:0;padding:24px;background:#f4efe8;font-family:Georgia,'Times New Roman',serif;color:#1f2933;">
  <table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%" style="max-width:760px;margin:0 auto;background:#fffdf8;border:1px solid #e8dcc9;border-radius:20px;">
    <tr>
      <td style="padding:32px 36px;background:#5a4530;background-image:linear-gradient(135deg,#112d32,#8f5f3f);color:#fff7ec;border-top-left-radius:20px;border-top-right-radius:20px;">
        <div style="font-size:12px;letter-spacing:0.18em;text-transform:uppercase;opacity:0.82;">GTrade Watchlist</div>
        <h1 style="margin:12px 0 8px;font-size:30px;line-height:1.1;">{{ .Title }}</h1>
        <p style="margin:0;font-size:14px;opacity:0.92;">Generated at {{ .Date }}</p>
      </td>
    </tr>
    <tr>
      <td style="padding:24px;">
        {{ range .Changes }}
        <table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%" style="border-bottom:1px solid #efe5d6;">
          <tr>
            <td width="96" valign="middle" align="center" style="padding:18px 18px 18px 0;width:96px;">
              <table role="presentation" cellpadding="0" cellspacing="0" border="0" width="96" height="96" style="width:96px;height:96px;background:#f0e4d3;border-radius:16px;">
                <tr>
                  <td align="center" valign="middle" width="96" height="96" style="width:96px;height:96px;border-radius:16px;">
                    {{ if .image_url }}<img src="{{ .image_url }}" alt="{{ .name }}" width="96" height="96" style="display:block;margin:0 auto;border:0;border-radius:16px;width:96px;height:96px;">{{ end }}
                  </td>
                </tr>
              </table>
            </td>
            <td valign="middle" style="padding:18px 0;">
              <div style="font-size:11px;letter-spacing:0.14em;text-transform:uppercase;color:#8b6b4b;">{{ .game }} &bull; {{ .source }} &bull; {{ .game_mode }}</div>
              <div style="margin:6px 0 8px;font-size:22px;font-weight:700;color:#1c2733;">{{ .name }}</div>
              <table role="presentation" cellpadding="0" cellspacing="0" border="0">
                <tr>
                  <td style="padding:0 24px 4px 0;font-size:14px;color:#3c4b5a;"><strong>Current:</strong> {{ .current_value }}</td>
                  <td style="padding:0 24px 4px 0;font-size:14px;color:#3c4b5a;"><strong>Previous:</strong> {{ .previous_value }}</td>
                </tr>
                <tr>
                  <td style="padding:0 24px 0 0;font-size:14px;color:#3c4b5a;"><strong>Change:</strong> {{ .delta }}</td>
                  <td style="padding:0 24px 0 0;font-size:14px;color:#3c4b5a;"><strong>Day:</strong> {{ .collected_on }}</td>
                </tr>
              </table>
            </td>
          </tr>
        </table>
        {{ end }}
      </td>
    </tr>
  </table>
</body>
</html>
`

const watchlistSnapshotEmailTemplate = `
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{ .Title }}</title>
</head>
<body style="margin:0;padding:24px;background:#f4efe8;font-family:Georgia,'Times New Roman',serif;color:#1f2933;">
  <table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%" style="max-width:760px;margin:0 auto;background:#fffdf8;border:1px solid #e8dcc9;border-radius:20px;">
    <tr>
      <td style="padding:32px 36px;background:#5a4530;background-image:linear-gradient(135deg,#112d32,#8f5f3f);color:#fff7ec;border-top-left-radius:20px;border-top-right-radius:20px;">
        <div style="font-size:12px;letter-spacing:0.18em;text-transform:uppercase;opacity:0.82;">GTrade Watchlist</div>
        <h1 style="margin:12px 0 8px;font-size:30px;line-height:1.1;">{{ .Title }}</h1>
        <p style="margin:0;font-size:14px;opacity:0.92;">Generated at {{ .Date }}</p>
      </td>
    </tr>
    <tr>
      <td style="padding:24px;">
        {{ range .Items }}
        <table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%" style="border-bottom:1px solid #efe5d6;">
          <tr>
            <td width="96" valign="middle" align="center" style="padding:18px 18px 18px 0;width:96px;">
              <table role="presentation" cellpadding="0" cellspacing="0" border="0" width="96" height="96" style="width:96px;height:96px;background:#f0e4d3;border-radius:16px;">
                <tr>
                  <td align="center" valign="middle" width="96" height="96" style="width:96px;height:96px;border-radius:16px;">
                    {{ if .image_url }}<img src="{{ .image_url }}" alt="{{ .name }}" width="96" height="96" style="display:block;margin:0 auto;border:0;border-radius:16px;width:96px;height:96px;">{{ end }}
                  </td>
                </tr>
              </table>
            </td>
            <td valign="middle" style="padding:18px 0;">
              <div style="font-size:11px;letter-spacing:0.14em;text-transform:uppercase;color:#8b6b4b;">{{ .game }} &bull; {{ .source }} &bull; {{ .game_mode }}</div>
              <div style="margin:6px 0 8px;font-size:22px;font-weight:700;color:#1c2733;">{{ .name }}</div>
              <table role="presentation" cellpadding="0" cellspacing="0" border="0">
                <tr>
                  <td style="padding:0 24px 0 0;font-size:14px;color:#3c4b5a;"><strong>Current price:</strong> {{ .current_value }}</td>
                  <td style="padding:0 24px 0 0;font-size:14px;color:#3c4b5a;"><strong>Day:</strong> {{ .collected_on }}</td>
                </tr>
              </table>
            </td>
          </tr>
        </table>
        {{ end }}
      </td>
    </tr>
  </table>
</body>
</html>
`
