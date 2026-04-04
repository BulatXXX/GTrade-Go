package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type OutboxRecord struct {
	ID                int64
	Recipient         string
	Subject           string
	HTMLBody          string
	TextBody          string
	Status            string
	Provider          string
	ProviderMessageID *string
	ErrorMessage      *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	SentAt            *time.Time
}

type NotificationRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse pg config: %w", err)
	}

	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pg pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

func (r *NotificationRepository) CreateOutboxRecord(
	ctx context.Context,
	recipient, subject, htmlBody, textBody, provider string,
) (*OutboxRecord, error) {
	query := `
		INSERT INTO notification_outbox (recipient, subject, html_body, text_body, status, provider)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, recipient, subject, html_body, text_body, status, provider, provider_message_id, error_message, created_at, updated_at, sent_at
	`

	var record OutboxRecord
	if err := r.pool.QueryRow(ctx, query, recipient, subject, htmlBody, textBody, "queued", provider).Scan(
		&record.ID,
		&record.Recipient,
		&record.Subject,
		&record.HTMLBody,
		&record.TextBody,
		&record.Status,
		&record.Provider,
		&record.ProviderMessageID,
		&record.ErrorMessage,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.SentAt,
	); err != nil {
		return nil, fmt.Errorf("insert notification_outbox: %w", err)
	}

	return &record, nil
}

func (r *NotificationRepository) MarkSent(ctx context.Context, id int64, providerMessageID string, sentAt time.Time) error {
	query := `
		UPDATE notification_outbox
		SET status = 'sent',
		    provider_message_id = $2,
		    sent_at = $3,
		    error_message = NULL,
		    updated_at = NOW()
		WHERE id = $1
	`
	if _, err := r.pool.Exec(ctx, query, id, providerMessageID, sentAt); err != nil {
		return fmt.Errorf("mark notification sent: %w", err)
	}
	return nil
}

func (r *NotificationRepository) MarkFailed(ctx context.Context, id int64, errMessage string) error {
	query := `
		UPDATE notification_outbox
		SET status = 'failed',
		    error_message = $2,
		    updated_at = NOW()
		WHERE id = $1
	`
	if _, err := r.pool.Exec(ctx, query, id, errMessage); err != nil {
		return fmt.Errorf("mark notification failed: %w", err)
	}
	return nil
}
