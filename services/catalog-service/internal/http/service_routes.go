package http

import (
	"github.com/gin-gonic/gin"
	"gtrade/services/catalog-service/internal/handler"
)

func registerServiceRoutes(r *gin.Engine, h *handler.Handler) {
	r.GET("/items", h.Placeholder("items_list"))
	r.GET("/items/:id", h.Placeholder("items_get"))
	r.GET("/items/search", h.Placeholder("items_search"))
	r.POST("/items/upsert", h.Placeholder("items_upsert"))
}
