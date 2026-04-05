ALTER TABLE user_profiles
    ADD COLUMN IF NOT EXISTS avatar_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS bio TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE user_profiles
SET updated_at = created_at
WHERE updated_at IS NULL;

ALTER TABLE watchlist_items
    ALTER COLUMN item_id TYPE TEXT USING item_id::TEXT;
