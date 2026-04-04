package http

import (
	"github.com/gin-gonic/gin"
	"gtrade/services/auth-service/internal/handler"
)

func registerServiceRoutes(r *gin.Engine, h *handler.Handler) {
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)
	r.POST("/refresh", h.Refresh)
	r.POST("/password/reset/request", h.RequestPasswordReset)
	r.POST("/password/reset/confirm", h.ConfirmPasswordReset)
	r.POST("/email/verify", h.EmailVerify)
}
