package http

import (
	"github.com/gin-gonic/gin"
	"gtrade/services/auth-service/internal/handler"
)

func registerServiceRoutes(r *gin.Engine, h *handler.Handler) {
	r.POST("/register", h.Placeholder("register"))
	r.POST("/login", h.Placeholder("login"))
	r.POST("/refresh", h.Placeholder("refresh"))
	r.POST("/password/reset/request", h.Placeholder("password_reset_request"))
	r.POST("/password/reset/confirm", h.Placeholder("password_reset_confirm"))
	r.POST("/email/verify", h.Placeholder("email_verify"))
}
