package service

import (
	"context"
	"testing"
	"time"

	authclient "gtrade/services/user-asset-service/internal/client/auth"
	"gtrade/services/user-asset-service/internal/client/catalog"
	notificationclient "gtrade/services/user-asset-service/internal/client/notification"
	"gtrade/services/user-asset-service/internal/repository"
)

type stubPriceAlertRepo struct {
	subscriptions             []repository.NotificationSubscription
	statesByWatchlist         map[int64][]repository.WatchlistNotificationState
	dispatchStateByUser       map[int64]*repository.UserNotificationDispatchState
	upsertedStates            []repository.WatchlistNotificationState
	upsertedDispatchUserIDs   []int64
	upsertedDispatchProcessed []time.Time
}

func (s *stubPriceAlertRepo) ListNotificationSubscriptions(ctx context.Context) ([]repository.NotificationSubscription, error) {
	return s.subscriptions, nil
}

func (s *stubPriceAlertRepo) ListWatchlistNotificationStates(ctx context.Context, watchlistItemID int64) ([]repository.WatchlistNotificationState, error) {
	return s.statesByWatchlist[watchlistItemID], nil
}

func (s *stubPriceAlertRepo) UpsertWatchlistNotificationState(ctx context.Context, state repository.WatchlistNotificationState) error {
	s.upsertedStates = append(s.upsertedStates, state)
	return nil
}

func (s *stubPriceAlertRepo) GetUserNotificationDispatchState(ctx context.Context, userID int64) (*repository.UserNotificationDispatchState, error) {
	return s.dispatchStateByUser[userID], nil
}

func (s *stubPriceAlertRepo) UpsertUserNotificationDispatchState(ctx context.Context, userID int64, processedOn time.Time, sentAt *time.Time) error {
	s.upsertedDispatchUserIDs = append(s.upsertedDispatchUserIDs, userID)
	s.upsertedDispatchProcessed = append(s.upsertedDispatchProcessed, processedOn)
	return nil
}

type stubPriceAlertCatalog struct {
	item    *catalog.Item
	history *catalog.PriceHistoryResponse
}

func (s stubPriceAlertCatalog) GetItem(ctx context.Context, id string) (*catalog.Item, error) {
	return s.item, nil
}

func (s stubPriceAlertCatalog) GetPriceHistory(ctx context.Context, id, gameMode string, limit int) (*catalog.PriceHistoryResponse, error) {
	return s.history, nil
}

type stubPriceAlertAuth struct {
	contact  *authclient.UserContact
	contacts []authclient.UserContact
}

func (s stubPriceAlertAuth) GetUserContact(ctx context.Context, userID int64) (*authclient.UserContact, error) {
	return s.contact, nil
}

func (s stubPriceAlertAuth) ListUserContacts(ctx context.Context, verifiedOnly bool) ([]authclient.UserContact, error) {
	return s.contacts, nil
}

type stubPriceAlertNotification struct {
	requests []notificationclient.SendEmailRequest
}

func (s *stubPriceAlertNotification) SendEmail(ctx context.Context, input notificationclient.SendEmailRequest) error {
	s.requests = append(s.requests, input)
	return nil
}

func TestDetectPriceAlertChanges_InitializesStateWithoutSending(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 9, 9, 0, 0, 0, time.UTC)
	changes, initStates, sendStates := detectPriceAlertChanges(
		repository.NotificationSubscription{WatchlistID: 10, UserID: 1, ItemID: "item-1"},
		&catalog.Item{ID: "item-1", Name: "Frost Prime Set", Game: "warframe", Source: "market"},
		[]catalog.PriceHistoryEntry{{
			ItemID:      "item-1",
			Source:      "warframe-market",
			Value:       125,
			Currency:    "PLAT",
			CollectedOn: "2026-05-09",
			CollectedAt: now,
		}},
		nil,
		now,
	)

	if len(changes) != 0 {
		t.Fatalf("changes len = %d, want 0", len(changes))
	}
	if len(initStates) != 1 {
		t.Fatalf("initStates len = %d, want 1", len(initStates))
	}
	if len(sendStates) != 0 {
		t.Fatalf("sendStates len = %d, want 0", len(sendStates))
	}
}

func TestPriceAlertService_RunCycleImmediateSendsEmailAndAdvancesState(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 9, 10, 30, 0, 0, time.UTC)
	yesterday := dayStartUTC(now.Add(-24 * time.Hour))
	previousValue := 100.0

	repo := &stubPriceAlertRepo{
		subscriptions: []repository.NotificationSubscription{{
			WatchlistID:          21,
			UserID:               7,
			ItemID:               "item-1",
			NotifyEnabled:        true,
			Currency:             "plat",
			NotificationsEnabled: true,
			NotificationMode:     "immediate",
			NotificationTime:     "09:00",
		}},
		statesByWatchlist: map[int64][]repository.WatchlistNotificationState{
			21: {{
				WatchlistItemID:         21,
				Source:                  "warframe-market",
				GameMode:                "",
				LastNotifiedCollectedOn: &yesterday,
				LastNotifiedValue:       &previousValue,
			}},
		},
		dispatchStateByUser: map[int64]*repository.UserNotificationDispatchState{},
	}

	catalogClient := stubPriceAlertCatalog{
		item: &catalog.Item{
			ID:       "item-1",
			Game:     "warframe",
			Source:   "market",
			Name:     "Frost Prime Set",
			Slug:     "frost_prime_set",
			ImageURL: "https://cdn.example.com/frost.png",
			IsActive: true,
		},
		history: &catalog.PriceHistoryResponse{
			ItemID: "item-1",
			History: []catalog.PriceHistoryEntry{
				{
					ItemID:      "item-1",
					Source:      "warframe-market",
					Value:       125,
					Currency:    "PLAT",
					CollectedOn: "2026-05-09",
					CollectedAt: now,
				},
				{
					ItemID:      "item-1",
					Source:      "warframe-market",
					Value:       100,
					Currency:    "PLAT",
					CollectedOn: "2026-05-08",
					CollectedAt: now.Add(-24 * time.Hour),
				},
			},
		},
	}

	notificationClient := &stubPriceAlertNotification{}
	svc := NewPriceAlertService(
		repo,
		catalogClient,
		stubPriceAlertAuth{
			contact: &authclient.UserContact{
				UserID:        7,
				Email:         "user@example.com",
				EmailVerified: true,
			},
		},
		notificationClient,
	)

	if err := svc.RunCycle(context.Background(), now); err != nil {
		t.Fatalf("RunCycle: %v", err)
	}

	if len(notificationClient.requests) != 1 {
		t.Fatalf("notification requests = %d, want 1", len(notificationClient.requests))
	}
	if notificationClient.requests[0].To != "user@example.com" {
		t.Fatalf("recipient = %q", notificationClient.requests[0].To)
	}
	if len(repo.upsertedStates) != 1 {
		t.Fatalf("upsertedStates len = %d, want 1", len(repo.upsertedStates))
	}
	if repo.upsertedStates[0].LastNotifiedValue == nil || *repo.upsertedStates[0].LastNotifiedValue != 125 {
		t.Fatalf("upserted state = %#v", repo.upsertedStates[0])
	}
	if repo.upsertedStates[0].LastNotificationSentAt == nil {
		t.Fatalf("expected LastNotificationSentAt to be set: %#v", repo.upsertedStates[0])
	}
}

func TestPriceAlertService_SendAdminMessageToAllVerifiedUsers(t *testing.T) {
	t.Parallel()

	notificationClient := &stubPriceAlertNotification{}
	svc := NewPriceAlertService(
		&stubPriceAlertRepo{},
		stubPriceAlertCatalog{},
		stubPriceAlertAuth{
			contacts: []authclient.UserContact{
				{UserID: 1, Email: "one@example.com", EmailVerified: true},
				{UserID: 2, Email: "two@example.com", EmailVerified: true},
			},
		},
		notificationClient,
	)

	result, err := svc.SendAdminMessage(context.Background(), 0, "System update", "<p>Hello</p>", "Hello")
	if err != nil {
		t.Fatalf("SendAdminMessage: %v", err)
	}
	if result.EmailsSent != 2 || len(notificationClient.requests) != 2 {
		t.Fatalf("result=%#v requests=%d", result, len(notificationClient.requests))
	}
}
