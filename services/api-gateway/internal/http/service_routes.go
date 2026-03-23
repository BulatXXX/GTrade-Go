package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gtrade/services/api-gateway/internal/handler"
)

func registerServiceRoutes(r *gin.Engine, _ *handler.Handler) {
	r.Use(JWTAuthStub())

	authGroup := r.Group("/api/auth")
	authGroup.Any("/*path", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"proxy": "auth-service", "status": "placeholder"})
	})

	usersGroup := r.Group("/api/users")
	usersGroup.Any("/*path", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"proxy": "user-asset-service", "status": "placeholder"})
	})

	itemsGroup := r.Group("/api/items")
	itemsGroup.Any("/*path", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"proxy": "catalog-service", "status": "placeholder"})
	})

	notifyGroup := r.Group("/api/notifications")
	notifyGroup.Any("/*path", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"proxy": "notification-service", "status": "placeholder"})
	})
}
