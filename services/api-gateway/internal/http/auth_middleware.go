package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func JWTAuthStub() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}

		c.Set("jwt_claims", gin.H{"sub": "stub-user"})
		c.Next()
	}
}
