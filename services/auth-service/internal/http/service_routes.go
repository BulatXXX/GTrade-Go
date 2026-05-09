package http

import (
	"github.com/gin-gonic/gin"
	httpmiddleware "github.com/singularity/gtrade/shared/httpmiddleware"
	"gtrade/services/auth-service/internal/handler"
)

func registerServiceRoutes(r *gin.Engine, h *handler.Handler, jwtSecret, internalToken string) {
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)
	r.POST("/refresh", h.Refresh)
	r.POST("/password/reset/request", h.RequestPasswordReset)
	r.POST("/password/reset/confirm", h.ConfirmPasswordReset)
	r.POST("/email/verify", h.EmailVerify)

	adminGroup := r.Group("/admin")
	adminGroup.Use(httpmiddleware.RequireJWT(jwtSecret))
	adminGroup.Use(httpmiddleware.RequireRole("admin"))
	adminGroup.GET("/users", h.ListUsers)
	adminGroup.PUT("/users/:id/role", h.UpdateUserRole)

	internalGroup := r.Group("/internal")
	internalGroup.Use(httpmiddleware.RequireInternalToken(internalToken))
	internalGroup.GET("/users/:id/email", h.GetInternalUserEmail)
	internalGroup.GET("/users/contacts", h.ListInternalUserContacts)
}
