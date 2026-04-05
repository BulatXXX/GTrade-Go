package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"gtrade/services/user-asset-service/internal/client/catalog"
	"gtrade/services/user-asset-service/internal/repository"
)

type stubRepo struct {
	createUserFn          func(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*repository.UserProfile, error)
	getUserFn             func(ctx context.Context, userID int64) (*repository.UserProfile, error)
	updateUserFn          func(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*repository.UserProfile, error)
	listWatchlistFn       func(ctx context.Context, userID int64) ([]repository.WatchlistItem, error)
	addWatchlistItemFn    func(ctx context.Context, userID int64, itemID string) (*repository.WatchlistItem, error)
	deleteWatchlistItemFn func(ctx context.Context, userID, watchlistID int64) (bool, error)
	listRecentFn          func(ctx context.Context, userID int64, limit int) ([]repository.WatchlistItem, error)
	getPreferencesFn      func(ctx context.Context, userID int64) (*repository.UserPreferences, error)
	upsertPreferencesFn   func(ctx context.Context, userID int64, currency string, notificationsEnabled bool) (*repository.UserPreferences, error)
}

func (s stubRepo) CreateUser(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*repository.UserProfile, error) {
	return s.createUserFn(ctx, userID, displayName, avatarURL, bio)
}
func (s stubRepo) GetUser(ctx context.Context, userID int64) (*repository.UserProfile, error) {
	return s.getUserFn(ctx, userID)
}
func (s stubRepo) UpdateUser(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*repository.UserProfile, error) {
	return s.updateUserFn(ctx, userID, displayName, avatarURL, bio)
}
func (s stubRepo) ListWatchlist(ctx context.Context, userID int64) ([]repository.WatchlistItem, error) {
	return s.listWatchlistFn(ctx, userID)
}
func (s stubRepo) AddWatchlistItem(ctx context.Context, userID int64, itemID string) (*repository.WatchlistItem, error) {
	return s.addWatchlistItemFn(ctx, userID, itemID)
}
func (s stubRepo) DeleteWatchlistItem(ctx context.Context, userID, watchlistID int64) (bool, error) {
	return s.deleteWatchlistItemFn(ctx, userID, watchlistID)
}
func (s stubRepo) ListRecent(ctx context.Context, userID int64, limit int) ([]repository.WatchlistItem, error) {
	return s.listRecentFn(ctx, userID, limit)
}
func (s stubRepo) GetPreferences(ctx context.Context, userID int64) (*repository.UserPreferences, error) {
	return s.getPreferencesFn(ctx, userID)
}
func (s stubRepo) UpsertPreferences(ctx context.Context, userID int64, currency string, notificationsEnabled bool) (*repository.UserPreferences, error) {
	return s.upsertPreferencesFn(ctx, userID, currency, notificationsEnabled)
}

type stubCatalog struct {
	getItemFn func(ctx context.Context, id string) (*catalog.Item, error)
}

func (s stubCatalog) GetItem(ctx context.Context, id string) (*catalog.Item, error) {
	return s.getItemFn(ctx, id)
}

func TestCreateUser_ValidatesAndTrimsProfileFields(t *testing.T) {
	t.Parallel()

	var gotDisplayName, gotAvatar, gotBio string
	svc := NewUserAssetService(stubRepo{
		createUserFn: func(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*repository.UserProfile, error) {
			gotDisplayName, gotAvatar, gotBio = displayName, avatarURL, bio
			return &repository.UserProfile{UserID: userID, DisplayName: displayName, AvatarURL: avatarURL, Bio: bio}, nil
		},
	}, stubCatalog{getItemFn: func(ctx context.Context, id string) (*catalog.Item, error) { return nil, nil }})

	_, err := svc.CreateUser(context.Background(), 1, "  Alice  ", " https://cdn/avatar.png ", " hi ")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if gotDisplayName != "Alice" || gotAvatar != "https://cdn/avatar.png" || gotBio != "hi" {
		t.Fatalf("trimmed values = %q %q %q", gotDisplayName, gotAvatar, gotBio)
	}
}

func TestAddWatchlistItem_UsesStringItemID(t *testing.T) {
	t.Parallel()

	var gotItemID string
	svc := NewUserAssetService(stubRepo{
		addWatchlistItemFn: func(ctx context.Context, userID int64, itemID string) (*repository.WatchlistItem, error) {
			gotItemID = itemID
			return &repository.WatchlistItem{ID: 1, UserID: userID, ItemID: itemID}, nil
		},
	}, stubCatalog{getItemFn: func(ctx context.Context, id string) (*catalog.Item, error) {
		return &catalog.Item{ID: id, IsActive: true}, nil
	}})

	_, err := svc.AddWatchlistItem(context.Background(), 42, "item-uuid-1")
	if err != nil {
		t.Fatalf("AddWatchlistItem: %v", err)
	}
	if gotItemID != "item-uuid-1" {
		t.Fatalf("itemID = %q", gotItemID)
	}
}

func TestGetPreferences_CreatesDefaultPreferences(t *testing.T) {
	t.Parallel()

	svc := NewUserAssetService(stubRepo{
		getPreferencesFn: func(ctx context.Context, userID int64) (*repository.UserPreferences, error) {
			return nil, nil
		},
		upsertPreferencesFn: func(ctx context.Context, userID int64, currency string, notificationsEnabled bool) (*repository.UserPreferences, error) {
			return &repository.UserPreferences{UserID: userID, Currency: currency, NotificationsEnabled: notificationsEnabled, UpdatedAt: time.Now()}, nil
		},
	}, stubCatalog{getItemFn: func(ctx context.Context, id string) (*catalog.Item, error) { return nil, nil }})

	prefs, err := svc.GetPreferences(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetPreferences: %v", err)
	}
	if prefs.Currency != "credits" || !prefs.NotificationsEnabled {
		t.Fatalf("prefs = %#v", prefs)
	}
}

func TestUpdateUser_ValidatesRequiredFields(t *testing.T) {
	t.Parallel()

	svc := NewUserAssetService(stubRepo{}, stubCatalog{getItemFn: func(ctx context.Context, id string) (*catalog.Item, error) { return nil, nil }})
	_, err := svc.UpdateUser(context.Background(), 0, "", "", "")
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestDeleteWatchlistItem_Delegates(t *testing.T) {
	t.Parallel()

	var gotUserID, gotWatchlistID int64
	svc := NewUserAssetService(stubRepo{
		deleteWatchlistItemFn: func(ctx context.Context, userID, watchlistID int64) (bool, error) {
			gotUserID, gotWatchlistID = userID, watchlistID
			return true, nil
		},
	}, stubCatalog{getItemFn: func(ctx context.Context, id string) (*catalog.Item, error) { return nil, nil }})

	deleted, err := svc.DeleteWatchlistItem(context.Background(), 5, 8)
	if err != nil {
		t.Fatalf("DeleteWatchlistItem: %v", err)
	}
	if !deleted || gotUserID != 5 || gotWatchlistID != 8 {
		t.Fatalf("delegation mismatch: deleted=%v user=%d watchlist=%d", deleted, gotUserID, gotWatchlistID)
	}
}

func TestGetUser_PropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	svc := NewUserAssetService(stubRepo{
		getUserFn: func(ctx context.Context, userID int64) (*repository.UserProfile, error) {
			return nil, errors.New("boom")
		},
	}, stubCatalog{getItemFn: func(ctx context.Context, id string) (*catalog.Item, error) { return nil, nil }})

	_, err := svc.GetUser(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAddWatchlistItem_FailsWhenCatalogItemMissing(t *testing.T) {
	t.Parallel()

	svc := NewUserAssetService(stubRepo{
		addWatchlistItemFn: func(ctx context.Context, userID int64, itemID string) (*repository.WatchlistItem, error) {
			t.Fatal("repository should not be called")
			return nil, nil
		},
	}, stubCatalog{
		getItemFn: func(ctx context.Context, id string) (*catalog.Item, error) {
			return nil, catalog.ErrNotFound
		},
	})

	_, err := svc.AddWatchlistItem(context.Background(), 42, "missing-item")
	if err == nil {
		t.Fatal("expected error")
	}
}
