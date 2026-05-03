ALTER TABLE prices
    ADD COLUMN IF NOT EXISTS game_mode TEXT NOT NULL DEFAULT '';

ALTER TABLE prices
    ADD COLUMN IF NOT EXISTS collected_on DATE;

UPDATE prices
SET collected_on = (collected_at AT TIME ZONE 'UTC')::DATE
WHERE collected_on IS NULL;

ALTER TABLE prices
    ALTER COLUMN collected_on SET DEFAULT CURRENT_DATE;

ALTER TABLE prices
    ALTER COLUMN collected_on SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_prices_item_source_mode_day
    ON prices(item_id, source, game_mode, collected_on);
