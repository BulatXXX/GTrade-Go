package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrDuplicate = errors.New("duplicate")

type UserProfile struct {
	UserID      int64
	DisplayName string
	AvatarURL   string
	Bio         string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type WatchlistItem struct {
	ID        int64
	UserID    int64
	ItemID    string
	CreatedAt time.Time
}

type UserPreferences struct {
	UserID               int64
	Currency             string
	NotificationsEnabled bool
	UpdatedAt            time.Time
}

type UserAssetRepository struct {
	pool *pgxpool.Pool
}

func NewUserAssetRepository(pool *pgxpool.Pool) *UserAssetRepository {
	return &UserAssetRepository{pool: pool}
}

func (r *UserAssetRepository) CreateUser(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*UserProfile, error) {
	query := `
		INSERT INTO user_profiles (external_user_id, display_name, avatar_url, bio, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING external_user_id, display_name, avatar_url, bio, created_at, updated_at
	`

	var p UserProfile
	if err := r.pool.QueryRow(ctx, query, userID, displayName, avatarURL, bio).Scan(&p.UserID, &p.DisplayName, &p.AvatarURL, &p.Bio, &p.CreatedAt, &p.UpdatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("create user profile: %w", err)
	}
	return &p, nil
}

func (r *UserAssetRepository) GetUser(ctx context.Context, userID int64) (*UserProfile, error) {
	query := `SELECT external_user_id, display_name, avatar_url, bio, created_at, updated_at FROM user_profiles WHERE external_user_id = $1`
	var p UserProfile
	if err := r.pool.QueryRow(ctx, query, userID).Scan(&p.UserID, &p.DisplayName, &p.AvatarURL, &p.Bio, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user profile: %w", err)
	}
	return &p, nil
}

func (r *UserAssetRepository) UpdateUser(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*UserProfile, error) {
	query := `
		UPDATE user_profiles
		SET display_name = $2, avatar_url = $3, bio = $4, updated_at = NOW()
		WHERE external_user_id = $1
		RETURNING external_user_id, display_name, avatar_url, bio, created_at, updated_at
	`

	var p UserProfile
	if err := r.pool.QueryRow(ctx, query, userID, displayName, avatarURL, bio).Scan(&p.UserID, &p.DisplayName, &p.AvatarURL, &p.Bio, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("update user profile: %w", err)
	}
	return &p, nil
}

func (r *UserAssetRepository) ListWatchlist(ctx context.Context, userID int64) ([]WatchlistItem, error) {
	query := `
		SELECT id, user_id, item_id, created_at
		FROM watchlist_items
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list watchlist: %w", err)
	}
	defer rows.Close()

	var items []WatchlistItem
	for rows.Next() {
		var it WatchlistItem
		if err := rows.Scan(&it.ID, &it.UserID, &it.ItemID, &it.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan watchlist row: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("watchlist rows: %w", err)
	}
	return items, nil
}

func (r *UserAssetRepository) AddWatchlistItem(ctx context.Context, userID int64, itemID string) (*WatchlistItem, error) {
	query := `
		INSERT INTO watchlist_items (user_id, item_id)
		VALUES ($1, $2)
		RETURNING id, user_id, item_id, created_at
	`
	var it WatchlistItem
	if err := r.pool.QueryRow(ctx, query, userID, itemID).Scan(&it.ID, &it.UserID, &it.ItemID, &it.CreatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("add watchlist item: %w", err)
	}
	return &it, nil
}

func (r *UserAssetRepository) DeleteWatchlistItem(ctx context.Context, userID, watchlistID int64) (bool, error) {
	query := `DELETE FROM watchlist_items WHERE id = $1 AND user_id = $2`
	res, err := r.pool.Exec(ctx, query, watchlistID, userID)
	if err != nil {
		return false, fmt.Errorf("delete watchlist item: %w", err)
	}
	return res.RowsAffected() > 0, nil
}

func (r *UserAssetRepository) ListRecent(ctx context.Context, userID int64, limit int) ([]WatchlistItem, error) {
	query := `
		SELECT id, user_id, item_id, created_at
		FROM watchlist_items
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := r.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent: %w", err)
	}
	defer rows.Close()

	var items []WatchlistItem
	for rows.Next() {
		var it WatchlistItem
		if err := rows.Scan(&it.ID, &it.UserID, &it.ItemID, &it.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan recent row: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recent rows: %w", err)
	}
	return items, nil
}

func (r *UserAssetRepository) GetPreferences(ctx context.Context, userID int64) (*UserPreferences, error) {
	query := `
		SELECT user_id, currency, notifications_enabled, updated_at
		FROM user_preferences
		WHERE user_id = $1
	`
	var p UserPreferences
	if err := r.pool.QueryRow(ctx, query, userID).Scan(&p.UserID, &p.Currency, &p.NotificationsEnabled, &p.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get preferences: %w", err)
	}
	return &p, nil
}

func (r *UserAssetRepository) UpsertPreferences(ctx context.Context, userID int64, currency string, notificationsEnabled bool) (*UserPreferences, error) {
	query := `
		INSERT INTO user_preferences (user_id, currency, notifications_enabled, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id)
		DO UPDATE SET currency = EXCLUDED.currency, notifications_enabled = EXCLUDED.notifications_enabled, updated_at = NOW()
		RETURNING user_id, currency, notifications_enabled, updated_at
	`
	var p UserPreferences
	if err := r.pool.QueryRow(ctx, query, userID, currency, notificationsEnabled).Scan(&p.UserID, &p.Currency, &p.NotificationsEnabled, &p.UpdatedAt); err != nil {
		return nil, fmt.Errorf("upsert preferences: %w", err)
	}
	return &p, nil
}
