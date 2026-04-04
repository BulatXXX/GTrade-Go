package http

import (
	"context"
	"encoding/json"
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
	"gtrade/services/auth-service/internal/handler"
	"gtrade/services/auth-service/internal/repository"
	"gtrade/services/auth-service/internal/service"
)

func TestRouterContract_PasswordResetRequestDoesNotExposeToken(t *testing.T) {
	ctx := context.Background()
	pool := newAuthHTTPTestPool(t, ctx)
	repo := repository.NewAuthRepository(pool)
	svc := service.NewAuthService(repo, testJWTSecret, service.NoopEmailNotifier{})

	email := "notify-password-reset@example.com"
	if _, err := svc.Register(ctx, email, "secret123"); err != nil {
		t.Fatalf("register: %v", err)
	}

	router := NewRouter(zerolog.Nop(), handler.New("auth-service", svc))
	req := newJSONRequest(t, http.MethodPost, "/password/reset/request", map[string]string{
		"email": email,
	})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	assertJSONFields(t, rec.Body.Bytes(), map[string]any{
		"status": "accepted",
	})
	assertJSONFieldMissing(t, rec.Body.Bytes(), "reset_token")

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM password_reset_tokens`).Scan(&count); err != nil {
		t.Fatalf("count password_reset_tokens: %v", err)
	}
	if count != 1 {
		t.Fatalf("password_reset_tokens count = %d, want 1", count)
	}
}

func TestRouterContract_EmailVerificationRequestDoesNotExposeToken(t *testing.T) {
	ctx := context.Background()
	pool := newAuthHTTPTestPool(t, ctx)
	repo := repository.NewAuthRepository(pool)
	svc := service.NewAuthService(repo, testJWTSecret, service.NoopEmailNotifier{})

	email := "notify-email-verify@example.com"
	if _, err := svc.Register(ctx, email, "secret123"); err != nil {
		t.Fatalf("register: %v", err)
	}

	router := NewRouter(zerolog.Nop(), handler.New("auth-service", svc))
	req := newJSONRequest(t, http.MethodPost, "/email/verify", map[string]string{
		"email": email,
	})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	assertJSONFields(t, rec.Body.Bytes(), map[string]any{
		"status": "verification_requested",
	})
	assertJSONFieldMissing(t, rec.Body.Bytes(), "verification_token")

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM email_verification_tokens`).Scan(&count); err != nil {
		t.Fatalf("count email_verification_tokens: %v", err)
	}
	if count != 1 {
		t.Fatalf("email_verification_tokens count = %d, want 1", count)
	}
}

func newAuthHTTPTestPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
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

	applyAuthHTTPMigrations(t, ctx, pool.Exec)

	if _, err := pool.Exec(ctx, "TRUNCATE TABLE email_verification_tokens, password_reset_tokens, refresh_tokens, users RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	return pool
}

func applyAuthHTTPMigrations(
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

func assertJSONFieldMissing(t *testing.T, body []byte, key string) {
	t.Helper()

	got := map[string]any{}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if _, ok := got[key]; ok {
		t.Fatalf("response unexpectedly contains %q: %s", key, string(body))
	}
}

const testJWTSecret = "http-contract-test-secret"
