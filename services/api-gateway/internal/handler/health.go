package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gtrade/services/api-gateway/internal/model"
)

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, model.HealthResponse{
		Status:  "ok",
		Service: h.serviceName,
	})
}
