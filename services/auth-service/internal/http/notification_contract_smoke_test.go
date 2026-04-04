package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"gtrade/services/auth-service/internal/handler"
	"gtrade/services/auth-service/internal/service"
)

func TestRouterContract_PasswordResetRequestHidesResetToken(t *testing.T) {
	t.Parallel()

	router := NewRouter(
		zerolog.Nop(),
		handler.New("auth-service", stubAuthService{
			registerFn: func(ctx context.Context, email, password string) (*service.TokenPair, error) {
				return nil, errors.New("unexpected call")
			},
			loginFn: func(ctx context.Context, email, password string) (*service.TokenPair, error) {
				return nil, errors.New("unexpected call")
			},
			refreshFn: func(ctx context.Context, refreshToken string) (*service.TokenPair, error) {
				return nil, errors.New("unexpected call")
			},
			requestPasswordResetFn: func(ctx context.Context, email string) (string, error) {
				return "reset-token", nil
			},
			confirmPasswordResetFn: func(ctx context.Context, token, newPassword string) error {
				return errors.New("unexpected call")
			},
			requestEmailVerificationFn: func(ctx context.Context, email string) (string, error) {
				return "", errors.New("unexpected call")
			},
			verifyEmailFn: func(ctx context.Context, token string) error {
				return errors.New("unexpected call")
			},
		}),
	)

	req := newJSONRequest(t, http.MethodPost, "/password/reset/request", map[string]string{
		"email": "user@example.com",
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
}

func TestRouterContract_EmailVerificationRequestHidesVerificationToken(t *testing.T) {
	t.Parallel()

	router := NewRouter(
		zerolog.Nop(),
		handler.New("auth-service", stubAuthService{
			registerFn: func(ctx context.Context, email, password string) (*service.TokenPair, error) {
				return nil, errors.New("unexpected call")
			},
			loginFn: func(ctx context.Context, email, password string) (*service.TokenPair, error) {
				return nil, errors.New("unexpected call")
			},
			refreshFn: func(ctx context.Context, refreshToken string) (*service.TokenPair, error) {
				return nil, errors.New("unexpected call")
			},
			requestPasswordResetFn: func(ctx context.Context, email string) (string, error) {
				return "", errors.New("unexpected call")
			},
			confirmPasswordResetFn: func(ctx context.Context, token, newPassword string) error {
				return errors.New("unexpected call")
			},
			requestEmailVerificationFn: func(ctx context.Context, email string) (string, error) {
				return "verification-token", nil
			},
			verifyEmailFn: func(ctx context.Context, token string) error {
				return errors.New("unexpected call")
			},
		}),
	)

	req := newJSONRequest(t, http.MethodPost, "/email/verify", map[string]string{
		"email": "user@example.com",
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
}
