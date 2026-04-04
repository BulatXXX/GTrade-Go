package service_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	t.Parallel()

	ctx := context.Background()
	pool := newTestPool(t, ctx)
	repo := repository.NewAuthRepository(pool)
	svc := service.NewAuthService(repo, testJWTSecret)

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
	t.Parallel()

	ctx := context.Background()
	pool := newTestPool(t, ctx)
	repo := repository.NewAuthRepository(pool)
	svc := service.NewAuthService(repo, testJWTSecret)

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

	if _, err := pool.Exec(ctx, "TRUNCATE TABLE refresh_tokens, users RESTART IDENTITY"); err != nil {
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

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file path")
	}

	migrationPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations", "0001_init.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read migration file: %v", err)
	}

	statements := strings.Split(string(migrationSQL), ";")
	for _, statement := range statements {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if _, err := execFn(ctx, statement); err != nil {
			t.Fatalf("apply migration statement %q: %v", statement, err)
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
