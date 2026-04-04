package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gtrade/services/catalog-service/internal/model"
	"gtrade/services/catalog-service/internal/repository"
	"gtrade/services/catalog-service/internal/service"
)

func (h *Handler) CreateItem(c *gin.Context) {
	var req model.CreateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.catalogService.CreateItem(c.Request.Context(), model.CreateItemInput{
		Game:         req.Game,
		Source:       req.Source,
		ExternalID:   req.ExternalID,
		Slug:         req.Slug,
		Name:         req.Name,
		Description:  req.Description,
		ImageURL:     req.ImageURL,
		Translations: req.Translations,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item input"})
		case errors.Is(err, repository.ErrItemExists):
			c.JSON(http.StatusConflict, gin.H{"error": "item already exists"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "create item failed"})
		}
		return
	}

	c.JSON(http.StatusCreated, model.ItemResponse{Item: *item})
}

func (h *Handler) UpsertItem(c *gin.Context) {
	var req model.CreateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.catalogService.UpsertItem(c.Request.Context(), model.CreateItemInput{
		Game:         req.Game,
		Source:       req.Source,
		ExternalID:   req.ExternalID,
		Slug:         req.Slug,
		Name:         req.Name,
		Description:  req.Description,
		ImageURL:     req.ImageURL,
		Translations: req.Translations,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item input"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "upsert item failed"})
		}
		return
	}

	c.JSON(http.StatusOK, model.ItemResponse{Item: *item})
}

func (h *Handler) UpdateItem(c *gin.Context) {
	var req model.UpdateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.catalogService.UpdateItem(c.Request.Context(), c.Param("id"), model.UpdateItemInput{
		Slug:         req.Slug,
		Name:         req.Name,
		Description:  req.Description,
		ImageURL:     req.ImageURL,
		IsActive:     req.IsActive,
		Translations: req.Translations,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item input"})
		case errors.Is(err, repository.ErrItemNotFound), errors.Is(err, service.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
		case errors.Is(err, repository.ErrItemExists), errors.Is(err, service.ErrConflict):
			c.JSON(http.StatusConflict, gin.H{"error": "item already exists"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "update item failed"})
		}
		return
	}

	c.JSON(http.StatusOK, model.ItemResponse{Item: *item})
}

func (h *Handler) DeleteItem(c *gin.Context) {
	if err := h.catalogService.DeleteItem(c.Request.Context(), c.Param("id")); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		case errors.Is(err, repository.ErrItemNotFound), errors.Is(err, service.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "delete item failed"})
		}
		return
	}

	c.JSON(http.StatusOK, model.DeleteItemResponse{Status: "deactivated"})
}

func (h *Handler) GetItemByID(c *gin.Context) {
	language := c.Query("language")
	item, err := h.catalogService.GetItemByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		case errors.Is(err, repository.ErrItemNotFound), errors.Is(err, service.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "get item failed"})
		}
		return
	}

	c.JSON(http.StatusOK, model.ItemResponse{Item: localizeItem(*item, language)})
}

func (h *Handler) ListItems(c *gin.Context) {
	language := c.Query("language")
	limit, err := parseIntQuery(c, "limit", 20)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
		return
	}
	offset, err := parseIntQuery(c, "offset", 0)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
		return
	}

	activeOnly, err := parseOptionalBoolQuery(c, "active_only")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid active_only"})
		return
	}

	items, err := h.catalogService.ListItems(c.Request.Context(), model.ListItemsFilter{
		Game:       c.Query("game"),
		Source:     c.Query("source"),
		ActiveOnly: activeOnly,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid list filter"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list items failed"})
		return
	}

	c.JSON(http.StatusOK, model.ListItemsResponse{
		Items:  localizeItems(items, language),
		Limit:  limit,
		Offset: offset,
	})
}

func (h *Handler) SearchItems(c *gin.Context) {
	language := c.Query("language")
	limit, err := parseIntQuery(c, "limit", 20)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
		return
	}
	offset, err := parseIntQuery(c, "offset", 0)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
		return
	}

	activeOnly, err := parseOptionalBoolQuery(c, "active_only")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid active_only"})
		return
	}

	items, err := h.catalogService.SearchItems(c.Request.Context(), model.SearchItemsFilter{
		Query:      c.Query("q"),
		Game:       c.Query("game"),
		Language:   language,
		ActiveOnly: activeOnly,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid search filter"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search items failed"})
		return
	}

	c.JSON(http.StatusOK, model.ListItemsResponse{
		Items:  localizeItems(items, language),
		Limit:  limit,
		Offset: offset,
	})
}

func parseIntQuery(c *gin.Context, key string, fallback int) (int, error) {
	value := c.Query(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func parseOptionalBoolQuery(c *gin.Context, key string) (*bool, error) {
	value := c.Query(key)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
