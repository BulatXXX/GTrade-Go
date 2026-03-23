package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gtrade/services/catalog-service/internal/model"
)

type Handler struct {
	serviceName string
}

func New(serviceName string) *Handler {
	return &Handler{serviceName: serviceName}
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, model.HealthResponse{
		Status:  "ok",
		Service: h.serviceName,
	})
}
