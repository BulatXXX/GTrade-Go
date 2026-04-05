package http

import (
	"github.com/gin-gonic/gin"
	httpmiddleware "github.com/singularity/gtrade/shared/httpmiddleware"
	"gtrade/services/api-gateway/internal/handler"
)

func registerServiceRoutes(r *gin.Engine, h *handler.Handler, jwtSecret string) {
	authGroup := r.Group("/api/auth")
	authGroup.Any("", h.ProxyAuth)
	authGroup.Any("/*path", h.ProxyAuth)

	usersGroup := r.Group("/api/users")
	usersGroup.Use(httpmiddleware.RequireJWT(jwtSecret))
	usersGroup.Any("", h.ProxyUsers)
	usersGroup.Any("/*path", h.ProxyUsers)

	itemsGroup := r.Group("/api/items")
	itemsGroup.Any("", h.ProxyCatalog)
	itemsGroup.Any("/*path", h.ProxyCatalog)

	marketGroup := r.Group("/api/market")
	marketGroup.Any("", h.ProxyMarket)
	marketGroup.Any("/*path", h.ProxyMarket)

	notifyGroup := r.Group("/api/notifications")
	notifyGroup.Use(httpmiddleware.RequireJWT(jwtSecret))
	notifyGroup.Any("", h.ProxyNotifications)
	notifyGroup.Any("/*path", h.ProxyNotifications)
}
