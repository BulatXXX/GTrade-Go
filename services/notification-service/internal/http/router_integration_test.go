package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"gtrade/services/notification-service/internal/handler"
	"gtrade/services/notification-service/internal/repository"
)

func TestSendEmailIntegration_PersistsOutboxRecord(t *testing.T) {
	ctx := context.Background()
	pool := newNotificationTestPool(t, ctx)

	router := NewRouter(zerolog.Nop(), handler.New("notification-service"))

	req := newNotificationJSONRequest(t, http.MethodPost, "/send-email", map[string]string{
		"to":        "user@example.com",
		"subject":   "Verify your email",
		"html_body": "<p>Verification link</p>",
		"text_body": "Verification link",
	})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM notification_outbox`).Scan(&count); err != nil {
		t.Fatalf("count notification_outbox: %v", err)
	}
	if count != 1 {
		t.Fatalf("notification_outbox count = %d, want 1", count)
	}

	var recipient, subject, provider, status string
	if err := pool.QueryRow(
		ctx,
		`SELECT recipient, subject, provider, status FROM notification_outbox ORDER BY id DESC LIMIT 1`,
	).Scan(&recipient, &subject, &provider, &status); err != nil {
		t.Fatalf("select outbox row: %v", err)
	}

	if recipient != "user@example.com" {
		t.Fatalf("recipient = %q, want %q", recipient, "user@example.com")
	}
	if subject != "Verify your email" {
		t.Fatalf("subject = %q, want %q", subject, "Verify your email")
	}
	if provider != "mock" {
		t.Fatalf("provider = %q, want %q", provider, "mock")
	}
	if status != "sent" {
		t.Fatalf("status = %q, want %q", status, "sent")
	}
}

func newNotificationTestPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	pool, err := repository.NewPostgresPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect test postgres: %v", err)
	}

	t.Cleanup(pool.Close)

	applyNotificationMigrations(t, ctx, pool.Exec)

	if _, err := pool.Exec(ctx, `TRUNCATE TABLE notification_outbox RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate notification_outbox: %v", err)
	}

	return pool
}

func applyNotificationMigrations(
	t *testing.T,
	ctx context.Context,
	execFn func(context.Context, string, ...any) (pgconn.CommandTag, error),
) {
	t.Helper()

	migrationPaths, err := filepath.Glob(filepath.Join("..", "..", "migrations", "*.sql"))
	if err != nil {
		t.Fatalf("glob migration files: %v", err)
	}
	sort.Strings(migrationPaths)

	for _, migrationPath := range migrationPaths {
		migrationSQL, err := os.ReadFile(migrationPath)
		if err != nil {
			t.Fatalf("read migration file %s: %v", migrationPath, err)
		}

		statements := strings.Split(string(migrationSQL), ";")
		for _, statement := range statements {
			statement = strings.TrimSpace(statement)
			if statement == "" {
				continue
			}
			if _, err := execFn(ctx, statement); err != nil {
				t.Fatalf("apply migration %s statement %q: %v", migrationPath, statement, err)
			}
		}
	}
}
