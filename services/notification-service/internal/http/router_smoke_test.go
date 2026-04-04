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
	"gtrade/services/notification-service/internal/handler"
	"gtrade/services/notification-service/internal/model"
)

type stubEmailService struct {
	sendEmailFn func(ctx context.Context, req model.SendEmailRequest) (*model.SendEmailResponse, error)
}

func (s stubEmailService) SendEmail(ctx context.Context, req model.SendEmailRequest) (*model.SendEmailResponse, error) {
	return s.sendEmailFn(ctx, req)
}

func TestRouterSmoke_SendEmailQueuesNotification(t *testing.T) {
	t.Parallel()

	router := NewRouter(zerolog.Nop(), handler.New("notification-service", stubEmailService{
		sendEmailFn: func(ctx context.Context, req model.SendEmailRequest) (*model.SendEmailResponse, error) {
			if req.To != "user@example.com" {
				return nil, errors.New("unexpected recipient")
			}
			return &model.SendEmailResponse{ID: 42, Status: "queued"}, nil
		},
	}))

	req := newNotificationJSONRequest(t, http.MethodPost, "/send-email", map[string]string{
		"to":        "user@example.com",
		"subject":   "Reset your password",
		"html_body": "<p>Reset link</p>",
		"text_body": "Reset link",
	})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
	}

	assertNotificationJSONFields(t, rec.Body.Bytes(), map[string]any{
		"id":     float64(42),
		"status": "queued",
	})
}

func TestRouterSmoke_SendEmailValidation(t *testing.T) {
	t.Parallel()

	router := NewRouter(zerolog.Nop(), handler.New("notification-service", stubEmailService{
		sendEmailFn: func(ctx context.Context, req model.SendEmailRequest) (*model.SendEmailResponse, error) {
			return nil, errors.New("to is required")
		},
	}))

	req := newNotificationJSONRequest(t, http.MethodPost, "/send-email", map[string]string{
		"subject":   "Reset your password",
		"html_body": "<p>Reset link</p>",
		"text_body": "Reset link",
	})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}

	assertNotificationHasError(t, rec.Body.Bytes())
}

func newNotificationJSONRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()

	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func assertNotificationJSONFields(t *testing.T, body []byte, want map[string]any) {
	t.Helper()

	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	for key, wantValue := range want {
		if got[key] != wantValue {
			t.Fatalf("response field %q = %#v, want %#v; body=%s", key, got[key], wantValue, string(body))
		}
	}
}

func assertNotificationHasError(t *testing.T, body []byte) {
	t.Helper()

	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if _, ok := got["error"]; !ok {
		t.Fatalf("expected error field in response: %v", got)
	}
}
