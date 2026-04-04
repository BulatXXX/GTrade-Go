package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAuthNotificationE2E_PasswordResetAndEmailVerificationCreateNotifications(t *testing.T) {
	authBaseURL := os.Getenv("AUTH_BASE_URL")
	notificationDBURL := os.Getenv("NOTIFICATION_TEST_DATABASE_URL")
	if authBaseURL == "" || notificationDBURL == "" {
		t.Skip("AUTH_BASE_URL or NOTIFICATION_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	notificationPool, err := pgxpool.New(ctx, notificationDBURL)
	if err != nil {
		t.Fatalf("connect notification postgres: %v", err)
	}
	defer notificationPool.Close()

	waitForHealth(t, authBaseURL+"/health")

	email := fmt.Sprintf("e2e-%d@example.com", time.Now().UnixNano())
	password := "secret123"

	registerResp := postJSON(t, authBaseURL+"/register", map[string]string{
		"email":    email,
		"password": password,
	})
	if registerResp.statusCode != http.StatusOK {
		t.Fatalf("register status = %d, want %d; body=%s", registerResp.statusCode, http.StatusOK, string(registerResp.body))
	}

	beforeCount := countOutboxRowsByRecipient(t, ctx, notificationPool, email)

	resetResp := postJSON(t, authBaseURL+"/password/reset/request", map[string]string{
		"email": email,
	})
	if resetResp.statusCode != http.StatusOK {
		t.Fatalf("password reset request status = %d, want %d; body=%s", resetResp.statusCode, http.StatusOK, string(resetResp.body))
	}
	assertJSONFieldEquals(t, resetResp.body, "status", "accepted")
	assertJSONFieldMissing(t, resetResp.body, "reset_token")

	waitForOutboxCount(t, ctx, notificationPool, email, beforeCount+1)
	assertLatestOutboxSubject(t, ctx, notificationPool, email, "Reset your password")

	verifyResp := postJSON(t, authBaseURL+"/email/verify", map[string]string{
		"email": email,
	})
	if verifyResp.statusCode != http.StatusOK {
		t.Fatalf("email verify request status = %d, want %d; body=%s", verifyResp.statusCode, http.StatusOK, string(verifyResp.body))
	}
	assertJSONFieldEquals(t, verifyResp.body, "status", "verification_requested")
	assertJSONFieldMissing(t, verifyResp.body, "verification_token")

	waitForOutboxCount(t, ctx, notificationPool, email, beforeCount+2)
	assertLatestOutboxSubject(t, ctx, notificationPool, email, "Verify your email")
}

type httpResponse struct {
	statusCode int
	body       []byte
}

func postJSON(t *testing.T, url string, payload any) httpResponse {
	t.Helper()

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytesReader(raw))
	if err != nil {
		t.Fatalf("post %s: %v", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	return httpResponse{statusCode: resp.StatusCode, body: body}
}

func bytesReader(raw []byte) *bytes.Reader {
	return bytes.NewReader(raw)
}

func waitForHealth(t *testing.T, url string) {
	t.Helper()

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("service did not become healthy: %s", url)
}

func countOutboxRowsByRecipient(t *testing.T, ctx context.Context, pool *pgxpool.Pool, recipient string) int {
	t.Helper()

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM notification_outbox WHERE recipient = $1`, recipient).Scan(&count); err != nil {
		t.Fatalf("count notification_outbox: %v", err)
	}
	return count
}

func waitForOutboxCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, recipient string, want int) {
	t.Helper()

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if got := countOutboxRowsByRecipient(t, ctx, pool, recipient); got >= want {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}

	got := countOutboxRowsByRecipient(t, ctx, pool, recipient)
	t.Fatalf("notification_outbox count = %d, want at least %d for recipient %s", got, want, recipient)
}

func assertLatestOutboxSubject(t *testing.T, ctx context.Context, pool *pgxpool.Pool, recipient, want string) {
	t.Helper()

	var subject string
	if err := pool.QueryRow(
		ctx,
		`SELECT subject FROM notification_outbox WHERE recipient = $1 ORDER BY id DESC LIMIT 1`,
		recipient,
	).Scan(&subject); err != nil {
		t.Fatalf("select latest notification subject: %v", err)
	}
	if subject != want {
		t.Fatalf("latest notification subject = %q, want %q", subject, want)
	}
}

func assertJSONFieldEquals(t *testing.T, body []byte, key string, want string) {
	t.Helper()

	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got[key] != want {
		t.Fatalf("response field %q = %#v, want %#v; body=%s", key, got[key], want, string(body))
	}
}

func assertJSONFieldMissing(t *testing.T, body []byte, key string) {
	t.Helper()

	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if _, ok := got[key]; ok {
		t.Fatalf("response unexpectedly contains %q: %s", key, string(body))
	}
}
