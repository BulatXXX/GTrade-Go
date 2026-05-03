CREATE TABLE IF NOT EXISTS items (
    id TEXT PRIMARY KEY,
    game TEXT NOT NULL,
    source TEXT NOT NULL,
    external_id TEXT NOT NULL,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    image_url TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (game, source, external_id),
    UNIQUE (game, source, slug)
);

CREATE TABLE IF NOT EXISTS item_translations (
    item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    language_code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    PRIMARY KEY (item_id, language_code)
);

CREATE TABLE IF NOT EXISTS prices (
    id BIGSERIAL PRIMARY KEY,
    item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    source TEXT NOT NULL,
    game_mode TEXT NOT NULL DEFAULT '',
    value NUMERIC(20, 6) NOT NULL,
    currency TEXT NOT NULL,
    collected_on DATE NOT NULL DEFAULT CURRENT_DATE,
    collected_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_prices_item_collected ON prices(item_id, collected_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS uq_prices_item_source_mode_day ON prices(item_id, source, game_mode, collected_on);
CREATE INDEX IF NOT EXISTS idx_items_game_source_name ON items(game, source, name);
CREATE INDEX IF NOT EXISTS idx_item_translations_item_id ON item_translations(item_id);
