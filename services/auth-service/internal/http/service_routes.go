package http

import (
	"github.com/gin-gonic/gin"
	httpmiddleware "github.com/singularity/gtrade/shared/httpmiddleware"
	"gtrade/services/auth-service/internal/handler"
)

func registerServiceRoutes(r *gin.Engine, h *handler.Handler, internalToken string) {
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)
	r.POST("/refresh", h.Refresh)
	r.POST("/password/reset/request", h.RequestPasswordReset)
	r.POST("/password/reset/confirm", h.ConfirmPasswordReset)
	r.POST("/email/verify", h.EmailVerify)

	internalGroup := r.Group("/internal")
	internalGroup.Use(httpmiddleware.RequireInternalToken(internalToken))
	internalGroup.GET("/users/:id/email", h.GetInternalUserEmail)
}
