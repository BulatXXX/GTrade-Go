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
}

type notificationEmailClient interface {
	SendEmail(ctx context.Context, input notificationclient.SendEmailRequest) error
}

type PriceAlertService struct {
	repo         priceAlertRepository
	catalog      catalogPriceClient
	auth         authContactClient
	notification notificationEmailClient
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

func NewPriceAlertService(repo priceAlertRepository, catalogClient catalogPriceClient, authClient authContactClient, notificationClient notificationEmailClient) *PriceAlertService {
	return &PriceAlertService{
		repo:         repo,
		catalog:      catalogClient,
		auth:         authClient,
		notification: notificationClient,
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
			continue
		}

		item, err := s.catalog.GetItem(ctx, sub.ItemID)
		if err != nil || item == nil || !item.IsActive {
			continue
		}

		historyResp, err := s.catalog.GetPriceHistory(ctx, sub.ItemID, "", 10)
		if err != nil || historyResp == nil {
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
		if err := s.dispatchUserNotification(ctx, pending, now, false); err != nil {
			return err
		}
	}

	for userID := range dueDigestUsers {
		pending := digest[userID]
		if pending != nil {
			if err := s.dispatchUserNotification(ctx, pending, now, true); err != nil {
				return err
			}
			continue
		}
		if err := s.repo.UpsertUserNotificationDispatchState(ctx, userID, dayStartUTC(now), nil); err != nil {
			return fmt.Errorf("mark empty digest processed: %w", err)
		}
	}

	return nil
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

func (s *PriceAlertService) dispatchUserNotification(ctx context.Context, pending *pendingUserNotification, now time.Time, markDigest bool) error {
	if pending == nil {
		return nil
	}

	sort.Slice(pending.changes, func(i, j int) bool {
		if pending.changes[i].CollectedAt.Equal(pending.changes[j].CollectedAt) {
			return pending.changes[i].ItemName < pending.changes[j].ItemName
		}
		return pending.changes[i].CollectedAt.After(pending.changes[j].CollectedAt)
	})

	contact, err := s.auth.GetUserContact(ctx, pending.userID)
	if err != nil || contact == nil || strings.TrimSpace(contact.Email) == "" || !contact.EmailVerified {
		for _, state := range pending.stateUpdates {
			if upsertErr := s.repo.UpsertWatchlistNotificationState(ctx, withoutSentAt(state)); upsertErr != nil {
				return fmt.Errorf("advance state without email: %w", upsertErr)
			}
		}
		if markDigest {
			if upsertErr := s.repo.UpsertUserNotificationDispatchState(ctx, pending.userID, dayStartUTC(now), nil); upsertErr != nil {
				return fmt.Errorf("mark digest processed without email: %w", upsertErr)
			}
		}
		return nil
	}

	if len(pending.changes) > 0 {
		htmlBody, textBody, subject, err := renderPriceAlertEmail(pending.mode, pending.changes, now)
		if err != nil {
			return fmt.Errorf("render price alert email: %w", err)
		}
		if err := s.notification.SendEmail(ctx, notificationclient.SendEmailRequest{
			To:       contact.Email,
			Subject:  subject,
			HTMLBody: htmlBody,
			TextBody: textBody,
		}); err != nil {
			return fmt.Errorf("send price alert email: %w", err)
		}
	}

	sentAt := now.UTC()
	for _, state := range pending.stateUpdates {
		state.LastNotificationSentAt = &sentAt
		if err := s.repo.UpsertWatchlistNotificationState(ctx, state); err != nil {
			return fmt.Errorf("upsert notification state after send: %w", err)
		}
	}
	if markDigest {
		if err := s.repo.UpsertUserNotificationDispatchState(ctx, pending.userID, dayStartUTC(now), &sentAt); err != nil {
			return fmt.Errorf("upsert digest dispatch state: %w", err)
		}
	}
	return nil
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

const priceAlertEmailTemplate = `
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{ .Title }}</title>
</head>
<body style="margin:0;padding:24px;background:#f4efe8;font-family:Georgia,'Times New Roman',serif;color:#1f2933;">
  <div style="max-width:760px;margin:0 auto;background:#fffdf8;border:1px solid #e8dcc9;border-radius:20px;overflow:hidden;">
    <div style="padding:32px 36px;background:linear-gradient(135deg,#112d32,#8f5f3f);color:#fff7ec;">
      <div style="font-size:12px;letter-spacing:0.18em;text-transform:uppercase;opacity:0.82;">GTrade Watchlist</div>
      <h1 style="margin:12px 0 8px;font-size:30px;line-height:1.1;">{{ .Title }}</h1>
      <p style="margin:0;font-size:14px;opacity:0.92;">Generated at {{ .Date }}</p>
    </div>
    <div style="padding:24px;">
      {{ range .Changes }}
      <div style="display:flex;gap:18px;padding:18px 0;border-bottom:1px solid #efe5d6;">
        <div style="width:96px;min-width:96px;height:96px;border-radius:16px;background:#f0e4d3;overflow:hidden;">
          {{ if .image_url }}
          <img src="{{ .image_url }}" alt="{{ .name }}" style="width:96px;height:96px;object-fit:cover;display:block;">
          {{ end }}
        </div>
        <div style="flex:1;">
          <div style="font-size:11px;letter-spacing:0.14em;text-transform:uppercase;color:#8b6b4b;">{{ .game }} • {{ .source }} • {{ .game_mode }}</div>
          <div style="margin:6px 0 8px;font-size:22px;font-weight:700;color:#1c2733;">{{ .name }}</div>
          <div style="display:flex;gap:24px;flex-wrap:wrap;font-size:14px;color:#3c4b5a;">
            <div><strong>Current:</strong> {{ .current_value }}</div>
            <div><strong>Previous:</strong> {{ .previous_value }}</div>
            <div><strong>Change:</strong> {{ .delta }}</div>
            <div><strong>Day:</strong> {{ .collected_on }}</div>
          </div>
        </div>
      </div>
      {{ end }}
    </div>
  </div>
</body>
</html>
`
