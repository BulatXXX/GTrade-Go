package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gtrade/services/api-integration-service/internal/model"
	"gtrade/services/api-integration-service/internal/service"
)

func (h *Handler) Search(c *gin.Context) {
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

	items, err := h.service.SearchItems(c.Request.Context(), model.SearchItemsQuery{
		Game:     c.Query("game"),
		GameMode: c.Query("game_mode"),
		Query:    c.Query("q"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		writeServiceError(c, err, "search items failed")
		return
	}

	c.JSON(http.StatusOK, model.SearchItemsResponse{
		Items:  items,
		Limit:  limit,
		Offset: offset,
	})
}

func (h *Handler) GetByID(c *gin.Context) {
	item, err := h.service.GetItem(c.Request.Context(), model.GetItemQuery{
		Game:     c.Query("game"),
		GameMode: c.Query("game_mode"),
		ID:       c.Param("id"),
	})
	if err != nil {
		writeServiceError(c, err, "get item failed")
		return
	}

	c.JSON(http.StatusOK, model.ItemResponse{Item: *item})
}

func (h *Handler) GetPricing(c *gin.Context) {
	price, err := h.service.GetPricing(c.Request.Context(), model.GetPricingQuery{
		Game:     c.Query("game"),
		GameMode: c.Query("game_mode"),
		ID:       c.Param("id"),
	})
	if err != nil {
		writeServiceError(c, err, "get price failed")
		return
	}

	c.JSON(http.StatusOK, model.PriceResponse{Price: *price})
}

func (h *Handler) GetTopPrice(c *gin.Context) {
	price, err := h.service.GetPricing(c.Request.Context(), model.GetPricingQuery{
		Game:     c.Query("game"),
		GameMode: c.Query("game_mode"),
		ID:       c.Param("id"),
	})
	if err != nil {
		writeServiceError(c, err, "get top price failed")
		return
	}

	c.JSON(http.StatusOK, model.TopPriceResponse{
		ItemID:    price.ItemID,
		Game:      price.Game,
		GameMode:  price.GameMode,
		Source:    price.Source,
		Currency:  price.Currency,
		Value:     price.Pricing.Current,
		FetchedAt: price.FetchedAt,
	})
}

func parseIntQuery(c *gin.Context, key string, fallback int) (int, error) {
	value := c.Query(key)
	if value == "" {
		return fallback, nil
	}
	return strconv.Atoi(value)
}

func writeServiceError(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
	case errors.Is(err, service.ErrUnsupportedGame):
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported game"})
	case errors.Is(err, service.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
	case errors.Is(err, service.ErrUpstreamFailed):
		c.JSON(http.StatusBadGateway, gin.H{"error": fallback})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": fallback})
	}
}
