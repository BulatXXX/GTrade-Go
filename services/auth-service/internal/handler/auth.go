package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

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
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		case errors.Is(err, service.ErrUserBlocked):
			c.JSON(http.StatusForbidden, gin.H{"error": "account is blocked"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		}
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

func (h *Handler) RequestPasswordReset(c *gin.Context) {
	var req model.PasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.authService.RequestPasswordReset(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "password reset request failed"})
		return
	}

	c.JSON(http.StatusOK, model.PasswordResetRequestResponse{
		Status: "accepted",
	})
}

func (h *Handler) ConfirmPasswordReset(c *gin.Context) {
	var req model.PasswordResetConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.authService.ConfirmPasswordReset(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		if errors.Is(err, service.ErrInvalidToken) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password reset token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "password reset confirm failed"})
		return
	}

	c.JSON(http.StatusOK, model.ActionStatusResponse{Status: "password_reset"})
}

func (h *Handler) EmailVerify(c *gin.Context) {
	var req model.EmailVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Token != "" {
		if err := h.authService.VerifyEmail(c.Request.Context(), req.Token); err != nil {
			if errors.Is(err, service.ErrInvalidToken) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid verification token"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "email verification failed"})
			return
		}

		c.JSON(http.StatusOK, model.EmailVerifyResponse{Status: "verified"})
		return
	}

	_, err := h.authService.RequestEmailVerification(c.Request.Context(), req.Email)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "email verification request failed"})
		return
	}

	c.JSON(http.StatusOK, model.EmailVerifyResponse{
		Status: "verification_requested",
	})
}

func (h *Handler) respondWithTokenPair(c *gin.Context, pair *service.TokenPair) {
	c.JSON(http.StatusOK, model.TokenPairResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    pair.ExpiresIn,
		Role:         pair.Role,
	})
}

func (h *Handler) GetInternalUserEmail(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	contact, err := h.authService.GetUserContact(c.Request.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "get user contact failed"})
		}
		return
	}

	c.JSON(http.StatusOK, model.InternalUserEmailResponse{
		UserID:        contact.UserID,
		Email:         contact.Email,
		EmailVerified: contact.EmailVerified,
	})
}

func (h *Handler) ListInternalUserContacts(c *gin.Context) {
	verifiedOnly := strings.EqualFold(strings.TrimSpace(c.Query("verified_only")), "true")

	users, err := h.authService.ListUserContacts(c.Request.Context(), verifiedOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list user contacts failed"})
		return
	}

	resp := model.InternalUserContactsResponse{
		Users: make([]model.InternalUserEmailResponse, 0, len(users)),
	}
	for _, user := range users {
		resp.Users = append(resp.Users, model.InternalUserEmailResponse{
			UserID:        user.UserID,
			Email:         user.Email,
			EmailVerified: user.EmailVerified,
		})
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListUsers(c *gin.Context) {
	users, err := h.authService.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list users failed"})
		return
	}

	resp := model.AdminUsersResponse{Users: make([]model.AdminUserResponse, 0, len(users))}
	for _, user := range users {
		resp.Users = append(resp.Users, model.AdminUserResponse{
			ID:            user.ID,
			Email:         user.Email,
			EmailVerified: user.EmailVerified,
			Role:          user.Role,
			Blocked:       user.Blocked,
			CreatedAt:     user.CreatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UpdateUserRole(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req model.UpdateUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.authService.UpdateUserRole(c.Request.Context(), userID, req.Role)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, model.AdminUserResponse{
		ID:            user.ID,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		Role:          user.Role,
		Blocked:       user.Blocked,
		CreatedAt:     user.CreatedAt.Format(time.RFC3339),
	})
}

func (h *Handler) SetUserBlocked(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req model.SetUserBlockedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.authService.SetUserBlocked(c.Request.Context(), userID, req.Blocked)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "set user blocked failed"})
		return
	}

	c.JSON(http.StatusOK, model.AdminUserResponse{
		ID:            user.ID,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		Role:          user.Role,
		Blocked:       user.Blocked,
		CreatedAt:     user.CreatedAt.Format(time.RFC3339),
	})
}
