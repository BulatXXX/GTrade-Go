package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	httpmiddleware "github.com/singularity/gtrade/shared/httpmiddleware"
	"gtrade/services/api-gateway/internal/handler"
)

func registerServiceRoutes(r *gin.Engine, _ *handler.Handler, jwtSecret string) {
	authGroup := r.Group("/api/auth")
	authGroup.Any("/*path", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"proxy": "auth-service", "status": "placeholder"})
	})

	usersGroup := r.Group("/api/users")
	usersGroup.Use(httpmiddleware.RequireJWT(jwtSecret))
	usersGroup.Any("/*path", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"proxy": "user-asset-service", "status": "placeholder"})
	})

	itemsGroup := r.Group("/api/items")
	itemsGroup.Use(httpmiddleware.RequireJWT(jwtSecret))
	itemsGroup.Any("/*path", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"proxy": "catalog-service", "status": "placeholder"})
	})

	notifyGroup := r.Group("/api/notifications")
	notifyGroup.Use(httpmiddleware.RequireJWT(jwtSecret))
	notifyGroup.Any("/*path", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"proxy": "notification-service", "status": "placeholder"})
	})
}
