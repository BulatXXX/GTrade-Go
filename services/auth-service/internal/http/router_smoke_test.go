package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"gtrade/services/auth-service/internal/handler"
	"gtrade/services/auth-service/internal/repository"
	"gtrade/services/auth-service/internal/service"
)

type stubAuthService struct {
	registerFn                 func(ctx context.Context, email, password string) (*service.TokenPair, error)
	loginFn                    func(ctx context.Context, email, password string) (*service.TokenPair, error)
	refreshFn                  func(ctx context.Context, refreshToken string) (*service.TokenPair, error)
	requestPasswordResetFn     func(ctx context.Context, email string) (string, error)
	confirmPasswordResetFn     func(ctx context.Context, token, newPassword string) error
	requestEmailVerificationFn func(ctx context.Context, email string) (string, error)
	verifyEmailFn              func(ctx context.Context, token string) error
}

func (s stubAuthService) Register(ctx context.Context, email, password string) (*service.TokenPair, error) {
	return s.registerFn(ctx, email, password)
}

func (s stubAuthService) Login(ctx context.Context, email, password string) (*service.TokenPair, error) {
	return s.loginFn(ctx, email, password)
}

func (s stubAuthService) Refresh(ctx context.Context, refreshToken string) (*service.TokenPair, error) {
	return s.refreshFn(ctx, refreshToken)
}

func (s stubAuthService) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	return s.requestPasswordResetFn(ctx, email)
}

func (s stubAuthService) ConfirmPasswordReset(ctx context.Context, token, newPassword string) error {
	return s.confirmPasswordResetFn(ctx, token, newPassword)
}

func (s stubAuthService) RequestEmailVerification(ctx context.Context, email string) (string, error) {
	return s.requestEmailVerificationFn(ctx, email)
}

func (s stubAuthService) VerifyEmail(ctx context.Context, token string) error {
	return s.verifyEmailFn(ctx, token)
}

func TestRouterSmoke(t *testing.T) {
	t.Parallel()

	tokenPair := &service.TokenPair{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresIn:    900,
	}

	router := NewRouter(
		zerolog.Nop(),
		handler.New("auth-service", stubAuthService{
			registerFn: func(ctx context.Context, email, password string) (*service.TokenPair, error) {
				if email == "exists@example.com" {
					return nil, repository.ErrEmailExists
				}
				return tokenPair, nil
			},
			loginFn: func(ctx context.Context, email, password string) (*service.TokenPair, error) {
				if email != "user@example.com" || password != "secret" {
					return nil, service.ErrInvalidCredentials
				}
				return tokenPair, nil
			},
			refreshFn: func(ctx context.Context, refreshToken string) (*service.TokenPair, error) {
				if refreshToken != "valid-refresh" {
					return nil, service.ErrInvalidToken
				}
				return tokenPair, nil
			},
			requestPasswordResetFn: func(ctx context.Context, email string) (string, error) {
				if email == "missing@example.com" {
					return "", nil
				}
				return "reset-token", nil
			},
			confirmPasswordResetFn: func(ctx context.Context, token, newPassword string) error {
				if token != "valid-reset-token" {
					return service.ErrInvalidToken
				}
				return nil
			},
			requestEmailVerificationFn: func(ctx context.Context, email string) (string, error) {
				if email == "missing@example.com" {
					return "", service.ErrUserNotFound
				}
				if email == "verified@example.com" {
					return "", nil
				}
				return "verification-token", nil
			},
			verifyEmailFn: func(ctx context.Context, token string) error {
				if token != "valid-verification-token" {
					return service.ErrInvalidToken
				}
				return nil
			},
		}),
	)

	tests := []struct {
		name           string
		method         string
		path           string
		body           any
		wantStatus     int
		wantJSONFields map[string]any
	}{
		{
			name:       "health",
			method:     http.MethodGet,
			path:       "/health",
			wantStatus: http.StatusOK,
			wantJSONFields: map[string]any{
				"status":  "ok",
				"service": "auth-service",
			},
		},
		{
			name:       "register success",
			method:     http.MethodPost,
			path:       "/register",
			body:       map[string]string{"email": "user@example.com", "password": "secret"},
			wantStatus: http.StatusOK,
			wantJSONFields: map[string]any{
				"access_token":  "access-token",
				"refresh_token": "refresh-token",
				"token_type":    "Bearer",
				"expires_in":    float64(900),
			},
		},
		{
			name:       "register conflict",
			method:     http.MethodPost,
			path:       "/register",
			body:       map[string]string{"email": "exists@example.com", "password": "secret"},
			wantStatus: http.StatusConflict,
			wantJSONFields: map[string]any{
				"error": "email already exists",
			},
		},
		{
			name:       "login success",
			method:     http.MethodPost,
			path:       "/login",
			body:       map[string]string{"email": "user@example.com", "password": "secret"},
			wantStatus: http.StatusOK,
			wantJSONFields: map[string]any{
				"access_token":  "access-token",
				"refresh_token": "refresh-token",
				"token_type":    "Bearer",
			},
		},
		{
			name:       "login unauthorized",
			method:     http.MethodPost,
			path:       "/login",
			body:       map[string]string{"email": "user@example.com", "password": "wrong"},
			wantStatus: http.StatusUnauthorized,
			wantJSONFields: map[string]any{
				"error": "invalid credentials",
			},
		},
		{
			name:       "refresh success",
			method:     http.MethodPost,
			path:       "/refresh",
			body:       map[string]string{"refresh_token": "valid-refresh"},
			wantStatus: http.StatusOK,
			wantJSONFields: map[string]any{
				"access_token":  "access-token",
				"refresh_token": "refresh-token",
				"token_type":    "Bearer",
			},
		},
		{
			name:       "refresh unauthorized",
			method:     http.MethodPost,
			path:       "/refresh",
			body:       map[string]string{"refresh_token": "bad-refresh"},
			wantStatus: http.StatusUnauthorized,
			wantJSONFields: map[string]any{
				"error": "invalid refresh token",
			},
		},
		{
			name:       "password reset request",
			method:     http.MethodPost,
			path:       "/password/reset/request",
			body:       map[string]string{"email": "user@example.com"},
			wantStatus: http.StatusOK,
			wantJSONFields: map[string]any{
				"status":      "accepted",
				"reset_token": "reset-token",
			},
		},
		{
			name:       "password reset confirm",
			method:     http.MethodPost,
			path:       "/password/reset/confirm",
			body:       map[string]string{"token": "valid-reset-token", "new_password": "secret2"},
			wantStatus: http.StatusOK,
			wantJSONFields: map[string]any{
				"status": "password_reset",
			},
		},
		{
			name:       "password reset confirm unauthorized",
			method:     http.MethodPost,
			path:       "/password/reset/confirm",
			body:       map[string]string{"token": "bad-token", "new_password": "secret2"},
			wantStatus: http.StatusUnauthorized,
			wantJSONFields: map[string]any{
				"error": "invalid password reset token",
			},
		},
		{
			name:       "email verify request",
			method:     http.MethodPost,
			path:       "/email/verify",
			body:       map[string]string{"email": "user@example.com"},
			wantStatus: http.StatusOK,
			wantJSONFields: map[string]any{
				"status":             "verification_requested",
				"verification_token": "verification-token",
			},
		},
		{
			name:       "email verify confirm",
			method:     http.MethodPost,
			path:       "/email/verify",
			body:       map[string]string{"token": "valid-verification-token"},
			wantStatus: http.StatusOK,
			wantJSONFields: map[string]any{
				"status": "verified",
			},
		},
		{
			name:       "email verify unauthorized",
			method:     http.MethodPost,
			path:       "/email/verify",
			body:       map[string]string{"token": "bad-verification-token"},
			wantStatus: http.StatusUnauthorized,
			wantJSONFields: map[string]any{
				"error": "invalid verification token",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := newJSONRequest(t, tt.method, tt.path, tt.body)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			assertJSONFields(t, rec.Body.Bytes(), tt.wantJSONFields)
		})
	}
}

func TestRouterSmoke_BadJSON(t *testing.T) {
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
				return "", errors.New("unexpected call")
			},
			verifyEmailFn: func(ctx context.Context, token string) error {
				return errors.New("unexpected call")
			},
		}),
	)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{"email":`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if _, ok := got["error"]; !ok {
		t.Fatalf("expected error field in response: %v", got)
	}
}

func newJSONRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()

	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func assertJSONFields(t *testing.T, body []byte, want map[string]any) {
	t.Helper()

	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	for key, wantValue := range want {
		if got[key] != wantValue {
			t.Fatalf("field %q = %#v, want %#v; body=%s", key, got[key], wantValue, string(body))
		}
	}
}
