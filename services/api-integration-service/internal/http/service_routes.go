package http

import (
	"github.com/gin-gonic/gin"
	"gtrade/services/api-integration-service/internal/handler"
)

func registerServiceRoutes(r *gin.Engine, h *handler.Handler) {
	r.GET("/search", h.Search)
	r.GET("/items/:id", h.GetByID)
	r.GET("/items/:id/prices", h.GetPricing)
	r.GET("/items/:id/top-price", h.GetTopPrice)
	r.POST("/internal/sync/item", h.SyncItem)
	r.POST("/internal/sync/search", h.SyncSearch)
}
