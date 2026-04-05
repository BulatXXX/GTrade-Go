package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"gtrade/services/api-gateway/internal/handler"
	"gtrade/services/api-gateway/internal/service"
)

type stubGatewayService struct {
	forwardFn func(ctx context.Context, target string, req service.ForwardRequest) (*service.ForwardResponse, error)
}

func (s stubGatewayService) Forward(ctx context.Context, target string, req service.ForwardRequest) (*service.ForwardResponse, error) {
	return s.forwardFn(ctx, target, req)
}

func TestRouterSmoke_ProxiesPublicRoutes(t *testing.T) {
	t.Parallel()

	router := NewRouter(zerolog.Nop(), handler.New("api-gateway", stubGatewayService{
		forwardFn: func(ctx context.Context, target string, req service.ForwardRequest) (*service.ForwardResponse, error) {
			switch target {
			case service.TargetAuth:
				if req.Path != "/login" {
					t.Fatalf("auth path = %q, want /login", req.Path)
				}
				return &service.ForwardResponse{StatusCode: http.StatusOK, Body: []byte(`{"service":"auth"}`)}, nil
			case service.TargetCatalog:
				if req.Path != "/items/search" || req.RawQuery != "q=frost&game=warframe" {
					t.Fatalf("catalog path/query = %q?%s", req.Path, req.RawQuery)
				}
				return &service.ForwardResponse{StatusCode: http.StatusOK, Body: []byte(`{"service":"catalog"}`)}, nil
			case service.TargetIntegration:
				if req.Path != "/items/frost_prime_set/prices" || req.RawQuery != "game=warframe" {
					t.Fatalf("market path/query = %q?%s", req.Path, req.RawQuery)
				}
				return &service.ForwardResponse{StatusCode: http.StatusOK, Body: []byte(`{"service":"integration"}`)}, nil
			default:
				t.Fatalf("unexpected target: %s", target)
				return nil, nil
			}
		},
	}), "test-secret")

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "auth", path: "/api/auth/login", want: `{"service":"auth"}`},
		{name: "catalog", path: "/api/items/search?q=frost&game=warframe", want: `{"service":"catalog"}`},
		{name: "market", path: "/api/market/items/frost_prime_set/prices?game=warframe", want: `{"service":"integration"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
			}
			if rec.Body.String() != tt.want {
				t.Fatalf("body = %s, want %s", rec.Body.String(), tt.want)
			}
		})
	}
}

func TestRouterSmoke_ProtectedRoutesRequireJWT(t *testing.T) {
	t.Parallel()

	router := NewRouter(zerolog.Nop(), handler.New("api-gateway", stubGatewayService{
		forwardFn: func(ctx context.Context, target string, req service.ForwardRequest) (*service.ForwardResponse, error) {
			return &service.ForwardResponse{StatusCode: http.StatusOK, Body: []byte(`{}`)}, nil
		},
	}), "test-secret")

	req := httptest.NewRequest(http.MethodGet, "/api/users/watchlist?user_id=user_1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestRouterSmoke_ProtectedRoutesProxyWithJWT(t *testing.T) {
	t.Parallel()

	router := NewRouter(zerolog.Nop(), handler.New("api-gateway", stubGatewayService{
		forwardFn: func(ctx context.Context, target string, req service.ForwardRequest) (*service.ForwardResponse, error) {
			if got := req.Headers.Get("Authorization"); got == "" {
				t.Fatal("authorization header must be forwarded")
			}
			switch target {
			case service.TargetUserAsset:
				if req.Path != "/watchlist" || req.RawQuery != "user_id=user_1" {
					t.Fatalf("user path/query = %q?%s", req.Path, req.RawQuery)
				}
				return &service.ForwardResponse{StatusCode: http.StatusOK, Body: []byte(`{"service":"user-asset"}`)}, nil
			case service.TargetNotification:
				if req.Path != "/send-email" {
					t.Fatalf("notification path = %q, want /send-email", req.Path)
				}
				return &service.ForwardResponse{StatusCode: http.StatusOK, Body: []byte(`{"service":"notification"}`)}, nil
			default:
				t.Fatalf("unexpected target: %s", target)
				return nil, nil
			}
		},
	}), "test-secret")

	token := signedToken(t, "test-secret")

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "users", path: "/api/users/watchlist?user_id=user_1", want: `{"service":"user-asset"}`},
		{name: "notifications", path: "/api/notifications/send-email", want: `{"service":"notification"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
			}
			if rec.Body.String() != tt.want {
				t.Fatalf("body = %s, want %s", rec.Body.String(), tt.want)
			}
		})
	}
}

func TestRouterSmoke_UpstreamFailureMapsToBadGateway(t *testing.T) {
	t.Parallel()

	router := NewRouter(zerolog.Nop(), handler.New("api-gateway", stubGatewayService{
		forwardFn: func(ctx context.Context, target string, req service.ForwardRequest) (*service.ForwardResponse, error) {
			return nil, service.ErrUpstream
		},
	}), "test-secret")

	req := httptest.NewRequest(http.MethodGet, "/api/auth/login", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadGateway, rec.Body.String())
	}
}

func signedToken(t *testing.T, secret string) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user_1",
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}
