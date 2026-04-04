package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gtrade/services/notification-service/internal/model"
)

func (h *Handler) SendEmail(c *gin.Context) {
	var req model.SendEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.emailService.SendEmail(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, resp)
}
