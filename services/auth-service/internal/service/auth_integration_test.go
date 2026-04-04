package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"gtrade/services/auth-service/internal/repository"
	"gtrade/services/auth-service/internal/service"
)

const testJWTSecret = "integration-test-secret"

func TestAuthServiceIntegration_RegisterLoginRefresh(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t, ctx)
	repo := repository.NewAuthRepository(pool)
	svc := service.NewAuthService(repo, testJWTSecret, service.NoopEmailNotifier{})

	email := fmt.Sprintf("user-%d@example.com", time.Now().UnixNano())
	password := "secret123"

	pair, err := svc.Register(ctx, email, password)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	assertTokenPair(t, pair)

	user, err := repo.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("get user by email: %v", err)
	}
	if user == nil {
		t.Fatal("expected user to be persisted")
	}
	if user.PasswordHash == password {
		t.Fatal("expected password to be stored hashed")
	}

	storedToken, err := repo.GetRefreshToken(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatalf("get refresh token after register: %v", err)
	}
	if storedToken == nil {
		t.Fatal("expected refresh token to be persisted")
	}
	if storedToken.UserID != user.ID {
		t.Fatalf("refresh token user_id = %d, want %d", storedToken.UserID, user.ID)
	}

	loginPair, err := svc.Login(ctx, email, password)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	assertTokenPair(t, loginPair)

	refreshedPair, err := svc.Refresh(ctx, loginPair.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	assertTokenPair(t, refreshedPair)

	oldRefresh, err := repo.GetRefreshToken(ctx, loginPair.RefreshToken)
	if err != nil {
		t.Fatalf("get old refresh token: %v", err)
	}
	if oldRefresh == nil || oldRefresh.RevokedAt == nil {
		t.Fatal("expected used refresh token to be revoked")
	}

	newRefresh, err := repo.GetRefreshToken(ctx, refreshedPair.RefreshToken)
	if err != nil {
		t.Fatalf("get new refresh token: %v", err)
	}
	if newRefresh == nil {
		t.Fatal("expected rotated refresh token to be persisted")
	}

	_, err = svc.Refresh(ctx, loginPair.RefreshToken)
	if !errors.Is(err, service.ErrInvalidToken) {
		t.Fatalf("refresh with revoked token error = %v, want %v", err, service.ErrInvalidToken)
	}
}

func TestAuthServiceIntegration_DuplicateRegisterAndInvalidLogin(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t, ctx)
	repo := repository.NewAuthRepository(pool)
	svc := service.NewAuthService(repo, testJWTSecret, service.NoopEmailNotifier{})

	email := fmt.Sprintf("duplicate-%d@example.com", time.Now().UnixNano())
	password := "secret123"

	if _, err := svc.Register(ctx, email, password); err != nil {
		t.Fatalf("initial register: %v", err)
	}

	_, err := svc.Register(ctx, email, password)
	if !errors.Is(err, repository.ErrEmailExists) {
		t.Fatalf("duplicate register error = %v, want %v", err, repository.ErrEmailExists)
	}

	_, err = svc.Login(ctx, email, "wrong-password")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("invalid login error = %v, want %v", err, service.ErrInvalidCredentials)
	}
}

func TestAuthServiceIntegration_PasswordResetAndEmailVerification(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t, ctx)
	repo := repository.NewAuthRepository(pool)
	notifier := &recordingNotifier{}
	svc := service.NewAuthService(repo, testJWTSecret, notifier)

	email := fmt.Sprintf("flow-%d@example.com", time.Now().UnixNano())
	oldPassword := "secret123"
	newPassword := "secret456"

	if _, err := svc.Register(ctx, email, oldPassword); err != nil {
		t.Fatalf("register: %v", err)
	}

	resetToken, err := svc.RequestPasswordReset(ctx, email)
	if err != nil {
		t.Fatalf("request password reset: %v", err)
	}
	if resetToken == "" {
		t.Fatal("expected password reset token to be returned")
	}
	if len(notifier.calls) != 1 {
		t.Fatalf("notification calls after password reset = %d, want 1", len(notifier.calls))
	}
	if notifier.calls[0].to != email {
		t.Fatalf("password reset notification recipient = %q, want %q", notifier.calls[0].to, email)
	}

	if err := svc.ConfirmPasswordReset(ctx, resetToken, newPassword); err != nil {
		t.Fatalf("confirm password reset: %v", err)
	}

	_, err = svc.Login(ctx, email, oldPassword)
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("login with old password error = %v, want %v", err, service.ErrInvalidCredentials)
	}

	if _, err := svc.Login(ctx, email, newPassword); err != nil {
		t.Fatalf("login with new password: %v", err)
	}

	verificationToken, err := svc.RequestEmailVerification(ctx, email)
	if err != nil {
		t.Fatalf("request email verification: %v", err)
	}
	if verificationToken == "" {
		t.Fatal("expected email verification token to be returned")
	}
	if len(notifier.calls) != 2 {
		t.Fatalf("notification calls after email verification request = %d, want 2", len(notifier.calls))
	}
	if notifier.calls[1].to != email {
		t.Fatalf("email verification notification recipient = %q, want %q", notifier.calls[1].to, email)
	}

	if err := svc.VerifyEmail(ctx, verificationToken); err != nil {
		t.Fatalf("verify email: %v", err)
	}

	user, err := repo.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("get user by email: %v", err)
	}
	if user == nil || !user.EmailVerified {
		t.Fatal("expected user email to be marked verified")
	}

	_, err = svc.RequestEmailVerification(ctx, "missing@example.com")
	if !errors.Is(err, service.ErrUserNotFound) {
		t.Fatalf("request email verification for missing user error = %v, want %v", err, service.ErrUserNotFound)
	}
}

func TestAuthServiceIntegration_NotificationClientCallsNotificationService(t *testing.T) {
	ctx := context.Background()
	pool := newTestPool(t, ctx)
	repo := repository.NewAuthRepository(pool)

	var received []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/send-email" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode notification request: %v", err)
		}
		received = append(received, body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"id":1,"status":"queued"}`))
	}))
	defer server.Close()

	svc := service.NewAuthService(repo, testJWTSecret, service.NewNotificationClient(server.URL))

	email := fmt.Sprintf("notify-http-%d@example.com", time.Now().UnixNano())
	if _, err := svc.Register(ctx, email, "secret123"); err != nil {
		t.Fatalf("register: %v", err)
	}

	if _, err := svc.RequestPasswordReset(ctx, email); err != nil {
		t.Fatalf("request password reset: %v", err)
	}
	if _, err := svc.RequestEmailVerification(ctx, email); err != nil {
		t.Fatalf("request email verification: %v", err)
	}

	if len(received) != 2 {
		t.Fatalf("notification requests = %d, want 2", len(received))
	}
	if received[0]["to"] != email {
		t.Fatalf("password reset to = %#v, want %#v", received[0]["to"], email)
	}
	if received[0]["subject"] != "Reset your password" {
		t.Fatalf("password reset subject = %#v", received[0]["subject"])
	}
	if received[1]["to"] != email {
		t.Fatalf("email verification to = %#v, want %#v", received[1]["to"], email)
	}
	if received[1]["subject"] != "Verify your email" {
		t.Fatalf("email verification subject = %#v", received[1]["subject"])
	}
}

func newTestPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
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

	applyAuthMigration(t, ctx, pool.Exec)

	if _, err := pool.Exec(ctx, "TRUNCATE TABLE email_verification_tokens, password_reset_tokens, refresh_tokens, users RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	return pool
}

func applyAuthMigration(
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

func assertTokenPair(t *testing.T, pair *service.TokenPair) {
	t.Helper()

	if pair == nil {
		t.Fatal("expected token pair, got nil")
	}
	if pair.AccessToken == "" {
		t.Fatal("expected access token to be present")
	}
	if pair.RefreshToken == "" {
		t.Fatal("expected refresh token to be present")
	}
	if pair.ExpiresIn <= 0 {
		t.Fatalf("expires_in = %d, want positive", pair.ExpiresIn)
	}
}

type recordingNotifier struct {
	calls []notificationCall
}

type notificationCall struct {
	to       string
	subject  string
	htmlBody string
	textBody string
}

func (n *recordingNotifier) SendEmail(ctx context.Context, to, subject, htmlBody, textBody string) error {
	n.calls = append(n.calls, notificationCall{
		to:       to,
		subject:  subject,
		htmlBody: htmlBody,
		textBody: textBody,
	})
	return nil
}
