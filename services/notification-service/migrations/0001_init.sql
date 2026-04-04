CREATE TABLE IF NOT EXISTS notification_outbox (
    id BIGSERIAL PRIMARY KEY,
    recipient TEXT NOT NULL,
    subject TEXT NOT NULL,
    html_body TEXT NOT NULL DEFAULT '',
    text_body TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    provider TEXT NOT NULL,
    provider_message_id TEXT,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMPTZ
);
