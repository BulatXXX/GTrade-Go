package http

import (
	"github.com/gin-gonic/gin"
	httpmiddleware "github.com/singularity/gtrade/shared/httpmiddleware"
	"gtrade/services/catalog-service/internal/handler"
)

func registerServiceRoutes(r *gin.Engine, h *handler.Handler, jwtSecret string) {
	r.GET("/items", h.ListItems)
	r.GET("/items/search", h.SearchItems)
	r.POST("/items", h.CreateItem)
	r.POST("/items/upsert", h.UpsertItem)
	r.GET("/items/:id", h.GetItemByID)
	r.GET("/items/:id/prices/history", h.GetPriceHistory)
	r.PUT("/items/:id", h.UpdateItem)
	r.DELETE("/items/:id", h.DeleteItem)

	adminGroup := r.Group("/admin")
	adminGroup.Use(httpmiddleware.RequireJWT(jwtSecret))
	adminGroup.Use(httpmiddleware.RequireRole("admin"))
	adminGroup.GET("/stats", h.GetStats)
	adminGroup.GET("/localizations/coverage", h.GetLocalizationCoverage)
	adminGroup.POST("/jobs/catalog-import", h.StartCatalogImport)
	adminGroup.POST("/jobs/price-history-sync", h.StartPriceHistorySync)
	adminGroup.GET("/jobs", h.ListAdminJobs)
	adminGroup.GET("/jobs/:id", h.GetAdminJob)
	adminGroup.GET("/scheduler-state", h.ListSchedulerStates)
}
