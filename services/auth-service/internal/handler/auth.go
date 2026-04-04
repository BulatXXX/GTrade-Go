package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gtrade/services/auth-service/internal/model"
	"gtrade/services/auth-service/internal/repository"
	"gtrade/services/auth-service/internal/service"
)

func (h *Handler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pair, err := h.authService.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrEmailExists):
			c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "register failed"})
		}
		return
	}

	h.respondWithTokenPair(c, pair)
}

func (h *Handler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pair, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}

	h.respondWithTokenPair(c, pair)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req model.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pair, err := h.authService.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "refresh failed"})
		return
	}

	h.respondWithTokenPair(c, pair)
}

func (h *Handler) Placeholder(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": h.serviceName,
			"action":  action,
			"status":  "not_implemented",
		})
	}
}

func (h *Handler) respondWithTokenPair(c *gin.Context, pair *service.TokenPair) {
	c.JSON(http.StatusOK, model.TokenPairResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    pair.ExpiresIn,
	})
}
