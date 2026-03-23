package http

import (
	"github.com/gin-gonic/gin"
	"gtrade/services/user-asset-service/internal/handler"
)

func registerServiceRoutes(r *gin.Engine, h *handler.Handler) {
	r.GET("/watchlist", h.Placeholder("watchlist_list"))
	r.POST("/watchlist", h.Placeholder("watchlist_create"))
	r.DELETE("/watchlist/:id", h.Placeholder("watchlist_delete"))
	r.GET("/recent", h.Placeholder("recent_list"))
	r.GET("/preferences", h.Placeholder("preferences_get"))
	r.PUT("/preferences", h.Placeholder("preferences_update"))
}
