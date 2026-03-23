CREATE TABLE IF NOT EXISTS items (
    id BIGSERIAL PRIMARY KEY,
    game TEXT NOT NULL,
    external_id TEXT NOT NULL,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    image_url TEXT,
    currency TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (game, external_id),
    UNIQUE (game, slug)
);

CREATE TABLE IF NOT EXISTS prices (
    id BIGSERIAL PRIMARY KEY,
    item_id BIGINT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    source TEXT NOT NULL,
    value NUMERIC(20, 6) NOT NULL,
    currency TEXT NOT NULL,
    collected_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_prices_item_collected ON prices(item_id, collected_at DESC);
CREATE INDEX IF NOT EXISTS idx_items_game_name ON items(game, name);
