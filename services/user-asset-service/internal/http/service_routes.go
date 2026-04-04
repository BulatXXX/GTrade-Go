package http

import (
	"github.com/gin-gonic/gin"
	"gtrade/services/user-asset-service/internal/handler"
)

func registerServiceRoutes(r *gin.Engine, h *handler.Handler) {
	r.POST("/users", h.CreateUser)
	r.GET("/users/:id", h.GetUser)

	r.GET("/watchlist", h.GetWatchlist)
	r.POST("/watchlist", h.CreateWatchlist)
	r.DELETE("/watchlist/:id", h.DeleteWatchlist)

	r.GET("/recent", h.GetRecent)

	r.GET("/preferences", h.GetPreferences)
	r.PUT("/preferences", h.UpdatePreferences)
}
