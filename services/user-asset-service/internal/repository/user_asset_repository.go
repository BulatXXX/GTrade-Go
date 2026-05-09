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
	ID            int64
	UserID        int64
	ItemID        string
	NotifyEnabled bool
	CreatedAt     time.Time
}

type UserPreferences struct {
	UserID               int64
	Currency             string
	NotificationsEnabled bool
	NotificationMode     string
	NotificationTime     string
	UpdatedAt            time.Time
}

type NotificationSubscription struct {
	WatchlistID          int64
	UserID               int64
	ItemID               string
	NotifyEnabled        bool
	Currency             string
	NotificationsEnabled bool
	NotificationMode     string
	NotificationTime     string
}

type WatchlistNotificationState struct {
	WatchlistItemID         int64
	Source                  string
	GameMode                string
	LastNotifiedCollectedOn *time.Time
	LastNotifiedValue       *float64
	LastNotificationSentAt  *time.Time
}

type UserNotificationDispatchState struct {
	UserID                int64
	LastDigestProcessedOn *time.Time
	LastDigestSentAt      *time.Time
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
		SELECT id, user_id, item_id, notify_enabled, created_at
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
		if err := rows.Scan(&it.ID, &it.UserID, &it.ItemID, &it.NotifyEnabled, &it.CreatedAt); err != nil {
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
		RETURNING id, user_id, item_id, notify_enabled, created_at
	`
	var it WatchlistItem
	if err := r.pool.QueryRow(ctx, query, userID, itemID).Scan(&it.ID, &it.UserID, &it.ItemID, &it.NotifyEnabled, &it.CreatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("add watchlist item: %w", err)
	}
	return &it, nil
}

func (r *UserAssetRepository) UpdateWatchlistItemNotification(ctx context.Context, userID, watchlistID int64, notifyEnabled bool) (*WatchlistItem, error) {
	query := `
		UPDATE watchlist_items
		SET notify_enabled = $3
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, item_id, notify_enabled, created_at
	`
	var it WatchlistItem
	if err := r.pool.QueryRow(ctx, query, watchlistID, userID, notifyEnabled).Scan(&it.ID, &it.UserID, &it.ItemID, &it.NotifyEnabled, &it.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("update watchlist item notification: %w", err)
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
		SELECT id, user_id, item_id, notify_enabled, created_at
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
		if err := rows.Scan(&it.ID, &it.UserID, &it.ItemID, &it.NotifyEnabled, &it.CreatedAt); err != nil {
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
		SELECT user_id, currency, notifications_enabled, notification_mode, notification_time, updated_at
		FROM user_preferences
		WHERE user_id = $1
	`
	var p UserPreferences
	if err := r.pool.QueryRow(ctx, query, userID).Scan(&p.UserID, &p.Currency, &p.NotificationsEnabled, &p.NotificationMode, &p.NotificationTime, &p.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get preferences: %w", err)
	}
	return &p, nil
}

func (r *UserAssetRepository) UpsertPreferences(ctx context.Context, userID int64, currency string, notificationsEnabled bool, notificationMode, notificationTime string) (*UserPreferences, error) {
	query := `
		INSERT INTO user_preferences (user_id, currency, notifications_enabled, notification_mode, notification_time, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (user_id)
		DO UPDATE SET
			currency = EXCLUDED.currency,
			notifications_enabled = EXCLUDED.notifications_enabled,
			notification_mode = EXCLUDED.notification_mode,
			notification_time = EXCLUDED.notification_time,
			updated_at = NOW()
		RETURNING user_id, currency, notifications_enabled, notification_mode, notification_time, updated_at
	`
	var p UserPreferences
	if err := r.pool.QueryRow(ctx, query, userID, currency, notificationsEnabled, notificationMode, notificationTime).Scan(&p.UserID, &p.Currency, &p.NotificationsEnabled, &p.NotificationMode, &p.NotificationTime, &p.UpdatedAt); err != nil {
		return nil, fmt.Errorf("upsert preferences: %w", err)
	}
	return &p, nil
}

func (r *UserAssetRepository) ListNotificationSubscriptions(ctx context.Context) ([]NotificationSubscription, error) {
	query := `
		SELECT
			w.id,
			w.user_id,
			w.item_id,
			w.notify_enabled,
			COALESCE(p.currency, 'credits'),
			COALESCE(p.notifications_enabled, TRUE),
			COALESCE(p.notification_mode, 'daily_digest'),
			COALESCE(p.notification_time, '09:00')
		FROM watchlist_items w
		LEFT JOIN user_preferences p ON p.user_id = w.user_id
		WHERE w.notify_enabled = TRUE
		ORDER BY w.user_id ASC, w.id ASC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list notification subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []NotificationSubscription
	for rows.Next() {
		var sub NotificationSubscription
		if err := rows.Scan(
			&sub.WatchlistID,
			&sub.UserID,
			&sub.ItemID,
			&sub.NotifyEnabled,
			&sub.Currency,
			&sub.NotificationsEnabled,
			&sub.NotificationMode,
			&sub.NotificationTime,
		); err != nil {
			return nil, fmt.Errorf("scan notification subscription: %w", err)
		}
		subs = append(subs, sub)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("notification subscription rows: %w", err)
	}
	return subs, nil
}

func (r *UserAssetRepository) ListWatchlistNotificationStates(ctx context.Context, watchlistItemID int64) ([]WatchlistNotificationState, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT watchlist_item_id, source, game_mode, last_notified_collected_on, last_notified_value, last_notification_sent_at
		FROM watchlist_notification_state
		WHERE watchlist_item_id = $1
	`, watchlistItemID)
	if err != nil {
		return nil, fmt.Errorf("list watchlist notification states: %w", err)
	}
	defer rows.Close()

	var states []WatchlistNotificationState
	for rows.Next() {
		var state WatchlistNotificationState
		var collectedOn *time.Time
		var lastValue *float64
		var sentAt *time.Time
		if err := rows.Scan(&state.WatchlistItemID, &state.Source, &state.GameMode, &collectedOn, &lastValue, &sentAt); err != nil {
			return nil, fmt.Errorf("scan watchlist notification state: %w", err)
		}
		state.LastNotifiedCollectedOn = collectedOn
		state.LastNotifiedValue = lastValue
		state.LastNotificationSentAt = sentAt
		states = append(states, state)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("watchlist notification state rows: %w", err)
	}
	return states, nil
}

func (r *UserAssetRepository) UpsertWatchlistNotificationState(ctx context.Context, state WatchlistNotificationState) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO watchlist_notification_state (
			watchlist_item_id,
			source,
			game_mode,
			last_notified_collected_on,
			last_notified_value,
			last_notification_sent_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (watchlist_item_id, source, game_mode)
		DO UPDATE SET
			last_notified_collected_on = EXCLUDED.last_notified_collected_on,
			last_notified_value = EXCLUDED.last_notified_value,
			last_notification_sent_at = EXCLUDED.last_notification_sent_at
	`, state.WatchlistItemID, state.Source, state.GameMode, state.LastNotifiedCollectedOn, state.LastNotifiedValue, state.LastNotificationSentAt)
	if err != nil {
		return fmt.Errorf("upsert watchlist notification state: %w", err)
	}
	return nil
}

func (r *UserAssetRepository) GetUserNotificationDispatchState(ctx context.Context, userID int64) (*UserNotificationDispatchState, error) {
	var state UserNotificationDispatchState
	var processedOn *time.Time
	var sentAt *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, last_digest_processed_on, last_digest_sent_at
		FROM user_notification_dispatch_state
		WHERE user_id = $1
	`, userID).Scan(&state.UserID, &processedOn, &sentAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user notification dispatch state: %w", err)
	}
	state.LastDigestProcessedOn = processedOn
	state.LastDigestSentAt = sentAt
	return &state, nil
}

func (r *UserAssetRepository) UpsertUserNotificationDispatchState(ctx context.Context, userID int64, processedOn time.Time, sentAt *time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_notification_dispatch_state (user_id, last_digest_processed_on, last_digest_sent_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id)
		DO UPDATE SET
			last_digest_processed_on = EXCLUDED.last_digest_processed_on,
			last_digest_sent_at = EXCLUDED.last_digest_sent_at
	`, userID, processedOn.UTC(), sentAt)
	if err != nil {
		return fmt.Errorf("upsert user notification dispatch state: %w", err)
	}
	return nil
}
