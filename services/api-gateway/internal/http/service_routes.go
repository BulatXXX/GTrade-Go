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

	adminGroup := r.Group("/api/admin")
	adminGroup.Use(httpmiddleware.RequireJWT(jwtSecret))
	adminGroup.Use(httpmiddleware.RequireRole("admin"))

	usersGroup := r.Group("/api/users")
	usersGroup.Use(httpmiddleware.RequireJWT(jwtSecret))
	usersGroup.Any("", h.ProxyUsers)
	usersGroup.Any("/*path", h.ProxyUsers)

	itemsGroup := r.Group("/api/items")
	itemsGroup.GET("", h.ProxyCatalog)
	itemsGroup.GET("/*path", h.ProxyCatalog)

	adminItemsGroup := r.Group("/api/items")
	adminItemsGroup.Use(httpmiddleware.RequireJWT(jwtSecret))
	adminItemsGroup.Use(httpmiddleware.RequireRole("admin"))
	adminItemsGroup.POST("", h.ProxyCatalog)
	adminItemsGroup.POST("/*path", h.ProxyCatalog)
	adminItemsGroup.PUT("/*path", h.ProxyCatalog)
	adminItemsGroup.DELETE("/*path", h.ProxyCatalog)

	adminAuthGroup := adminGroup.Group("/auth")
	adminAuthGroup.Any("", h.ProxyAuthAdmin)
	adminAuthGroup.Any("/*path", h.ProxyAuthAdmin)

	adminCatalogGroup := adminGroup.Group("/catalog")
	adminCatalogGroup.Any("", h.ProxyCatalogAdmin)
	adminCatalogGroup.Any("/*path", h.ProxyCatalogAdmin)

	adminUserAssetsGroup := adminGroup.Group("/user-assets")
	adminUserAssetsGroup.Any("", h.ProxyUsersAdmin)
	adminUserAssetsGroup.Any("/*path", h.ProxyUsersAdmin)

	marketGroup := r.Group("/api/market")
	marketGroup.Any("", h.ProxyMarket)
	marketGroup.Any("/*path", h.ProxyMarket)

	notifyGroup := r.Group("/api/notifications")
	notifyGroup.Use(httpmiddleware.RequireJWT(jwtSecret))
	notifyGroup.Any("", h.ProxyNotifications)
	notifyGroup.Any("/*path", h.ProxyNotifications)
}
