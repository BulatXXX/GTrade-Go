package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gtrade/services/user-asset-service/internal/model"
	"gtrade/services/user-asset-service/internal/repository"
)

func (h *Handler) CreateUser(c *gin.Context) {
	var req model.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	profile, err := h.userAssetService.CreateUser(c.Request.Context(), req.UserID, req.DisplayName, req.AvatarURL, req.Bio)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			c.JSON(http.StatusConflict, gin.H{"error": "user already exists"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, toUserProfileResponse(*profile))
}

func (h *Handler) GetUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	profile, err := h.userAssetService.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user"})
		return
	}
	if profile == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	watchlist, err := h.userAssetService.ListWatchlist(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch watchlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":      toUserProfileResponse(*profile),
		"watchlist": h.toWatchlistResponse(c.Request.Context(), watchlist),
	})
}

func (h *Handler) UpdateUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req model.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	profile, err := h.userAssetService.UpdateUser(c.Request.Context(), userID, req.DisplayName, req.AvatarURL, req.Bio)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if profile == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, toUserProfileResponse(*profile))
}

func (h *Handler) GetWatchlist(c *gin.Context) {
	userID, ok := parseUserIDQuery(c)
	if !ok {
		return
	}

	items, err := h.userAssetService.ListWatchlist(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list watchlist"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": h.toWatchlistResponse(c.Request.Context(), items)})
}

func (h *Handler) CreateWatchlist(c *gin.Context) {
	var req model.AddWatchlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.userAssetService.AddWatchlistItem(c.Request.Context(), req.UserID, req.ItemID)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			c.JSON(http.StatusConflict, gin.H{"error": "watchlist item already exists"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, h.toWatchlistItemResponse(c.Request.Context(), *item))
}

func (h *Handler) DeleteWatchlist(c *gin.Context) {
	userID, ok := parseUserIDQuery(c)
	if !ok {
		return
	}

	watchlistID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || watchlistID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid watchlist id"})
		return
	}

	deleted, err := h.userAssetService.DeleteWatchlistItem(c.Request.Context(), userID, watchlistID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !deleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "watchlist item not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) GetRecent(c *gin.Context) {
	userID, ok := parseUserIDQuery(c)
	if !ok {
		return
	}

	items, err := h.userAssetService.ListRecent(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list recent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": h.toWatchlistResponse(c.Request.Context(), items)})
}

func (h *Handler) GetPreferences(c *gin.Context) {
	userID, ok := parseUserIDQuery(c)
	if !ok {
		return
	}

	prefs, err := h.userAssetService.GetPreferences(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get preferences"})
		return
	}

	c.JSON(http.StatusOK, model.PreferencesResponse{
		UserID:               prefs.UserID,
		Currency:             prefs.Currency,
		NotificationsEnabled: prefs.NotificationsEnabled,
		UpdatedAt:            prefs.UpdatedAt.Format(timeFormat),
	})
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req model.UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prefs, err := h.userAssetService.UpdatePreferences(c.Request.Context(), req.UserID, req.Currency, req.NotificationsEnabled)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.PreferencesResponse{
		UserID:               prefs.UserID,
		Currency:             prefs.Currency,
		NotificationsEnabled: prefs.NotificationsEnabled,
		UpdatedAt:            prefs.UpdatedAt.Format(timeFormat),
	})
}

const timeFormat = "2006-01-02T15:04:05Z07:00"

func parseUserIDQuery(c *gin.Context) (int64, bool) {
	userID, err := strconv.ParseInt(c.Query("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid user_id query param is required"})
		return 0, false
	}
	return userID, true
}

func (h *Handler) toWatchlistResponse(ctx context.Context, items []repository.WatchlistItem) []model.WatchlistItemResponse {
	out := make([]model.WatchlistItemResponse, 0, len(items))
	for _, it := range items {
		out = append(out, h.toWatchlistItemResponse(ctx, it))
	}
	return out
}

func (h *Handler) toWatchlistItemResponse(ctx context.Context, it repository.WatchlistItem) model.WatchlistItemResponse {
	resp := model.WatchlistItemResponse{
		ID:        it.ID,
		UserID:    it.UserID,
		ItemID:    it.ItemID,
		CreatedAt: it.CreatedAt.Format(timeFormat),
	}
	if item, err := h.userAssetService.GetCatalogItem(ctx, it.ItemID); err == nil && item != nil {
		resp.Item = &model.CatalogItemSummary{
			ID:       item.ID,
			Game:     item.Game,
			Source:   item.Source,
			Name:     item.Name,
			Slug:     item.Slug,
			ImageURL: item.ImageURL,
		}
	}
	return resp
}

func toUserProfileResponse(profile repository.UserProfile) model.UserProfileResponse {
	return model.UserProfileResponse{
		UserID:      profile.UserID,
		DisplayName: profile.DisplayName,
		AvatarURL:   profile.AvatarURL,
		Bio:         profile.Bio,
		CreatedAt:   profile.CreatedAt.Format(timeFormat),
		UpdatedAt:   profile.UpdatedAt.Format(timeFormat),
	}
}
