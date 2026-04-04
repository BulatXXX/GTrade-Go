package httpmiddleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
)

func TestRequestIDGeneratesAndReturnsHeader(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/health", func(c *gin.Context) {
		requestID, exists := c.Get(ContextRequestID)
		if !exists {
			t.Fatal("request_id missing in context")
		}
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get(HeaderRequestID) == "" {
		t.Fatal("expected request id header to be set")
	}
}

func TestRequireJWT(t *testing.T) {
	t.Parallel()

	const secret = "test-secret"

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(RequestID(), RequestLogger(zerolog.Nop()))
	protected := r.Group("/protected")
	protected.Use(RequireJWT(secret))
	protected.GET("", func(c *gin.Context) {
		claims, exists := c.Get(ContextJWTClaims)
		if !exists {
			t.Fatal("jwt claims missing in context")
		}
		c.JSON(http.StatusOK, claims)
	})

	t.Run("unauthorized without header", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("authorized with valid token", func(t *testing.T) {
		t.Parallel()
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "user-1",
			"exp": time.Now().Add(time.Minute).Unix(),
		})
		signed, err := token.SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("sign token: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+signed)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})
}
