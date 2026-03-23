CREATE TABLE IF NOT EXISTS watchlist_items (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    item_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, item_id)
);

-- Вариант database-per-service: внешние ключи на users/items не задаются,
-- целостность по user_id/item_id обеспечивается на уровне сервисов.
