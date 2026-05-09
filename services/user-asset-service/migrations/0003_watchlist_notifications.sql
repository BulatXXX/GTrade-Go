ALTER TABLE watchlist_items
    ADD COLUMN IF NOT EXISTS notify_enabled BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE user_preferences
    ADD COLUMN IF NOT EXISTS notification_mode TEXT NOT NULL DEFAULT 'daily_digest',
    ADD COLUMN IF NOT EXISTS notification_time TEXT NOT NULL DEFAULT '09:00';

CREATE TABLE IF NOT EXISTS watchlist_notification_state (
    watchlist_item_id BIGINT NOT NULL REFERENCES watchlist_items(id) ON DELETE CASCADE,
    source TEXT NOT NULL,
    game_mode TEXT NOT NULL DEFAULT '',
    last_notified_collected_on DATE,
    last_notified_value NUMERIC(20, 6),
    last_notification_sent_at TIMESTAMPTZ,
    PRIMARY KEY (watchlist_item_id, source, game_mode)
);

CREATE TABLE IF NOT EXISTS user_notification_dispatch_state (
    user_id BIGINT PRIMARY KEY,
    last_digest_processed_on DATE,
    last_digest_sent_at TIMESTAMPTZ
);
