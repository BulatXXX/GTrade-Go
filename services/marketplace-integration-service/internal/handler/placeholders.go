package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Search(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"service": h.serviceName, "action": "search", "status": "not_implemented"})
}

func (h *Handler) GetByID(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"service": h.serviceName, "action": "get_item", "id": c.Param("id"), "status": "not_implemented"})
}

func (h *Handler) GetTopPrice(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"service": h.serviceName, "action": "top_price", "id": c.Param("id"), "status": "not_implemented"})
}
