package http

import (
	"github.com/gin-gonic/gin"
	httpmiddleware "github.com/singularity/gtrade/shared/httpmiddleware"
	"gtrade/services/api-integration-service/internal/handler"
)

func registerServiceRoutes(r *gin.Engine, h *handler.Handler, internalToken string) {
	r.GET("/search", h.Search)
	r.GET("/items/:id", h.GetByID)
	r.GET("/items/:id/prices", h.GetPricing)
	r.GET("/items/:id/top-price", h.GetTopPrice)

	internalGroup := r.Group("/internal")
	internalGroup.Use(httpmiddleware.RequireInternalToken(internalToken))
	internalGroup.POST("/sync/item", h.SyncItem)
	internalGroup.POST("/sync/search", h.SyncSearch)
}
