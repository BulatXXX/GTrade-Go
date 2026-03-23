package http

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"gtrade/services/api-gateway/internal/handler"
)

func NewRouter(logger zerolog.Logger, h *handler.Handler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(RequestLogger(logger))

	r.GET("/health", h.Health)
	registerServiceRoutes(r, h)

	return r
}
