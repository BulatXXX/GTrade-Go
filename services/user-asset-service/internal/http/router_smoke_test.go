package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"gtrade/services/user-asset-service/internal/handler"
	"gtrade/services/user-asset-service/internal/repository"
)

type stubUserAssetService struct {
	createUserFn          func(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*repository.UserProfile, error)
	getUserFn             func(ctx context.Context, userID int64) (*repository.UserProfile, error)
	updateUserFn          func(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*repository.UserProfile, error)
	listWatchlistFn       func(ctx context.Context, userID int64) ([]repository.WatchlistItem, error)
	addWatchlistItemFn    func(ctx context.Context, userID int64, itemID string) (*repository.WatchlistItem, error)
	deleteWatchlistItemFn func(ctx context.Context, userID, watchlistID int64) (bool, error)
	listRecentFn          func(ctx context.Context, userID int64) ([]repository.WatchlistItem, error)
	getPreferencesFn      func(ctx context.Context, userID int64) (*repository.UserPreferences, error)
	updatePreferencesFn   func(ctx context.Context, userID int64, currency string, notificationsEnabled bool) (*repository.UserPreferences, error)
}

func (s stubUserAssetService) CreateUser(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*repository.UserProfile, error) {
	return s.createUserFn(ctx, userID, displayName, avatarURL, bio)
}
func (s stubUserAssetService) GetUser(ctx context.Context, userID int64) (*repository.UserProfile, error) {
	return s.getUserFn(ctx, userID)
}
func (s stubUserAssetService) UpdateUser(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*repository.UserProfile, error) {
	return s.updateUserFn(ctx, userID, displayName, avatarURL, bio)
}
func (s stubUserAssetService) ListWatchlist(ctx context.Context, userID int64) ([]repository.WatchlistItem, error) {
	return s.listWatchlistFn(ctx, userID)
}
func (s stubUserAssetService) AddWatchlistItem(ctx context.Context, userID int64, itemID string) (*repository.WatchlistItem, error) {
	return s.addWatchlistItemFn(ctx, userID, itemID)
}
func (s stubUserAssetService) DeleteWatchlistItem(ctx context.Context, userID, watchlistID int64) (bool, error) {
	return s.deleteWatchlistItemFn(ctx, userID, watchlistID)
}
func (s stubUserAssetService) ListRecent(ctx context.Context, userID int64) ([]repository.WatchlistItem, error) {
	return s.listRecentFn(ctx, userID)
}
func (s stubUserAssetService) GetPreferences(ctx context.Context, userID int64) (*repository.UserPreferences, error) {
	return s.getPreferencesFn(ctx, userID)
}
func (s stubUserAssetService) UpdatePreferences(ctx context.Context, userID int64, currency string, notificationsEnabled bool) (*repository.UserPreferences, error) {
	return s.updatePreferencesFn(ctx, userID, currency, notificationsEnabled)
}

func TestRouterSmoke_UserAssetFlows(t *testing.T) {
	t.Parallel()

	now := time.Unix(0, 0).UTC()
	router := NewRouter(zerolog.Nop(), handler.New("user-asset-service", stubUserAssetService{
		createUserFn: func(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*repository.UserProfile, error) {
			return &repository.UserProfile{UserID: userID, DisplayName: displayName, AvatarURL: avatarURL, Bio: bio, CreatedAt: now, UpdatedAt: now}, nil
		},
		getUserFn: func(ctx context.Context, userID int64) (*repository.UserProfile, error) {
			return &repository.UserProfile{UserID: userID, DisplayName: "Alice", AvatarURL: "https://cdn/avatar.png", Bio: "bio", CreatedAt: now, UpdatedAt: now}, nil
		},
		updateUserFn: func(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*repository.UserProfile, error) {
			return &repository.UserProfile{UserID: userID, DisplayName: displayName, AvatarURL: avatarURL, Bio: bio, CreatedAt: now, UpdatedAt: now}, nil
		},
		listWatchlistFn: func(ctx context.Context, userID int64) ([]repository.WatchlistItem, error) {
			return []repository.WatchlistItem{{ID: 1, UserID: userID, ItemID: "item-1", CreatedAt: now}}, nil
		},
		addWatchlistItemFn: func(ctx context.Context, userID int64, itemID string) (*repository.WatchlistItem, error) {
			return &repository.WatchlistItem{ID: 2, UserID: userID, ItemID: itemID, CreatedAt: now}, nil
		},
		deleteWatchlistItemFn: func(ctx context.Context, userID, watchlistID int64) (bool, error) {
			return true, nil
		},
		listRecentFn: func(ctx context.Context, userID int64) ([]repository.WatchlistItem, error) {
			return []repository.WatchlistItem{{ID: 3, UserID: userID, ItemID: "item-2", CreatedAt: now}}, nil
		},
		getPreferencesFn: func(ctx context.Context, userID int64) (*repository.UserPreferences, error) {
			return &repository.UserPreferences{UserID: userID, Currency: "usd", NotificationsEnabled: true, UpdatedAt: now}, nil
		},
		updatePreferencesFn: func(ctx context.Context, userID int64, currency string, notificationsEnabled bool) (*repository.UserPreferences, error) {
			return &repository.UserPreferences{UserID: userID, Currency: currency, NotificationsEnabled: notificationsEnabled, UpdatedAt: now}, nil
		},
	}))

	tests := []struct {
		name       string
		method     string
		path       string
		body       any
		wantStatus int
		wantField  string
	}{
		{"health", http.MethodGet, "/health", nil, http.StatusOK, "status"},
		{"create user", http.MethodPost, "/users", map[string]any{"user_id": 1, "display_name": "Alice", "avatar_url": "https://cdn/avatar.png", "bio": "bio"}, http.StatusCreated, "user_id"},
		{"get user", http.MethodGet, "/users/1", nil, http.StatusOK, "user"},
		{"update user", http.MethodPut, "/users/1", map[string]any{"display_name": "Alice 2", "avatar_url": "https://cdn/2.png", "bio": "new"}, http.StatusOK, "display_name"},
		{"get watchlist", http.MethodGet, "/watchlist?user_id=1", nil, http.StatusOK, "items"},
		{"create watchlist", http.MethodPost, "/watchlist", map[string]any{"user_id": 1, "item_id": "item-1"}, http.StatusCreated, "item_id"},
		{"get recent", http.MethodGet, "/recent?user_id=1", nil, http.StatusOK, "items"},
		{"get preferences", http.MethodGet, "/preferences?user_id=1", nil, http.StatusOK, "currency"},
		{"update preferences", http.MethodPut, "/preferences", map[string]any{"user_id": 1, "currency": "eur", "notifications_enabled": false}, http.StatusOK, "currency"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newJSONRequest(t, tt.method, tt.path, tt.body)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			var got map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if _, ok := got[tt.wantField]; !ok {
				t.Fatalf("missing field %q in %v", tt.wantField, got)
			}
		})
	}
}

func TestRouterSmoke_ConflictOnDuplicateWatchlist(t *testing.T) {
	t.Parallel()

	router := NewRouter(zerolog.Nop(), handler.New("user-asset-service", stubUserAssetService{
		createUserFn: func(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*repository.UserProfile, error) {
			return nil, nil
		},
		getUserFn: func(ctx context.Context, userID int64) (*repository.UserProfile, error) { return nil, nil },
		updateUserFn: func(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*repository.UserProfile, error) {
			return nil, nil
		},
		listWatchlistFn: func(ctx context.Context, userID int64) ([]repository.WatchlistItem, error) { return nil, nil },
		addWatchlistItemFn: func(ctx context.Context, userID int64, itemID string) (*repository.WatchlistItem, error) {
			return nil, repository.ErrDuplicate
		},
		deleteWatchlistItemFn: func(ctx context.Context, userID, watchlistID int64) (bool, error) { return false, nil },
		listRecentFn:          func(ctx context.Context, userID int64) ([]repository.WatchlistItem, error) { return nil, nil },
		getPreferencesFn:      func(ctx context.Context, userID int64) (*repository.UserPreferences, error) { return nil, nil },
		updatePreferencesFn: func(ctx context.Context, userID int64, currency string, notificationsEnabled bool) (*repository.UserPreferences, error) {
			return nil, nil
		},
	}))

	req := newJSONRequest(t, http.MethodPost, "/watchlist", map[string]any{"user_id": 1, "item_id": "item-1"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func newJSONRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()
	if body == nil {
		return httptest.NewRequest(method, path, nil)
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestRouterSmoke_NotFoundUser(t *testing.T) {
	t.Parallel()

	router := NewRouter(zerolog.Nop(), handler.New("user-asset-service", stubUserAssetService{
		createUserFn: func(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*repository.UserProfile, error) {
			return nil, nil
		},
		getUserFn: func(ctx context.Context, userID int64) (*repository.UserProfile, error) { return nil, nil },
		updateUserFn: func(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*repository.UserProfile, error) {
			return nil, nil
		},
		listWatchlistFn: func(ctx context.Context, userID int64) ([]repository.WatchlistItem, error) { return nil, nil },
		addWatchlistItemFn: func(ctx context.Context, userID int64, itemID string) (*repository.WatchlistItem, error) {
			return nil, nil
		},
		deleteWatchlistItemFn: func(ctx context.Context, userID, watchlistID int64) (bool, error) { return false, nil },
		listRecentFn:          func(ctx context.Context, userID int64) ([]repository.WatchlistItem, error) { return nil, nil },
		getPreferencesFn: func(ctx context.Context, userID int64) (*repository.UserPreferences, error) {
			return nil, errors.New("unexpected")
		},
		updatePreferencesFn: func(ctx context.Context, userID int64, currency string, notificationsEnabled bool) (*repository.UserPreferences, error) {
			return nil, nil
		},
	}))

	req := httptest.NewRequest(http.MethodGet, "/users/99", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}
