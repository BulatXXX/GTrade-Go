package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type SendEmailRequest struct {
	To       string `json:"to"`
	Subject  string `json:"subject"`
	HTMLBody string `json:"html_body"`
	TextBody string `json:"text_body"`
}

func (h *Handler) SendEmail(c *gin.Context) {
	var req SendEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"service": h.serviceName,
		"action":  "send_email",
		"status":  "queued_placeholder",
		"to":      req.To,
	})
}
