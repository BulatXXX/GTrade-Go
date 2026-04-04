package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"gtrade/services/notification-service/internal/model"
)

type EmailUseCase interface {
	SendEmail(ctx context.Context, req model.SendEmailRequest) (*model.SendEmailResponse, error)
}

type Handler struct {
	serviceName  string
	emailService EmailUseCase
}

func New(serviceName string, emailService EmailUseCase) *Handler {
	return &Handler{serviceName: serviceName, emailService: emailService}
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, model.HealthResponse{
		Status:  "ok",
		Service: h.serviceName,
	})
}
