package http

import (
	"github.com/gin-gonic/gin"
	"gtrade/services/notification-service/internal/handler"
)

func registerServiceRoutes(r *gin.Engine, h *handler.Handler) {
	r.POST("/send-email", h.SendEmail)
}
