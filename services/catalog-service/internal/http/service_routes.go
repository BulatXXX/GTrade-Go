package http

import (
	"github.com/gin-gonic/gin"
	"gtrade/services/catalog-service/internal/handler"
)

func registerServiceRoutes(r *gin.Engine, h *handler.Handler) {
	r.GET("/items", h.ListItems)
	r.GET("/items/search", h.SearchItems)
	r.POST("/items", h.CreateItem)
	r.POST("/items/upsert", h.UpsertItem)
	r.GET("/items/:id", h.GetItemByID)
	r.GET("/items/:id/prices/history", h.GetPriceHistory)
	r.PUT("/items/:id", h.UpdateItem)
	r.DELETE("/items/:id", h.DeleteItem)
}
