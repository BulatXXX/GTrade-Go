package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Placeholder(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": h.serviceName,
			"action":  action,
			"status":  "not_implemented",
		})
	}
}
