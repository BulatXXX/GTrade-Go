package http

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	httpmiddleware "github.com/singularity/gtrade/shared/httpmiddleware"
	"gtrade/services/user-asset-service/internal/handler"
)

func NewRouter(logger zerolog.Logger, h *handler.Handler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(httpmiddleware.RequestID())
	r.Use(httpmiddleware.RequestLogger(logger))

	r.GET("/health", h.Health)
	registerServiceRoutes(r, h)

	return r
}
