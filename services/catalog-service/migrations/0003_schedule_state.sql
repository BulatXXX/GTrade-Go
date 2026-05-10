CREATE TABLE IF NOT EXISTS schedule_state (
    job_name         TEXT PRIMARY KEY,
    status           TEXT NOT NULL DEFAULT 'idle',
    last_started_at  TIMESTAMPTZ,
    last_finished_at TIMESTAMPTZ,
    last_error       TEXT,
    last_processed   INTEGER NOT NULL DEFAULT 0,
    last_total       INTEGER NOT NULL DEFAULT 0,
    runs_total       BIGINT NOT NULL DEFAULT 0,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO schedule_state (job_name)
VALUES
    ('price_history_sync'),
    ('catalog_import_warframe'),
    ('catalog_import_eve'),
    ('catalog_import_tarkov')
ON CONFLICT (job_name) DO NOTHING;
