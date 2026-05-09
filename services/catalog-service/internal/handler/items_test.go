package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gtrade/services/catalog-service/internal/model"
	"gtrade/services/catalog-service/internal/repository"
	"gtrade/services/catalog-service/internal/service"
)

type stubCatalogUseCase struct {
	getPriceHistoryFn func(ctx context.Context, itemID string, filter model.PriceHistoryFilter) ([]model.PriceHistoryEntry, error)
}

func (s stubCatalogUseCase) CreateItem(ctx context.Context, input model.CreateItemInput) (*model.Item, error) {
	return nil, errors.New("not implemented")
}

func (s stubCatalogUseCase) UpsertItem(ctx context.Context, input model.CreateItemInput) (*model.Item, error) {
	return nil, errors.New("not implemented")
}

func (s stubCatalogUseCase) UpdateItem(ctx context.Context, id string, input model.UpdateItemInput) (*model.Item, error) {
	return nil, errors.New("not implemented")
}

func (s stubCatalogUseCase) DeleteItem(ctx context.Context, id string) error {
	return errors.New("not implemented")
}

func (s stubCatalogUseCase) GetItemByID(ctx context.Context, id string) (*model.Item, error) {
	return nil, errors.New("not implemented")
}

func (s stubCatalogUseCase) ListItems(ctx context.Context, filter model.ListItemsFilter) ([]model.Item, error) {
	return nil, errors.New("not implemented")
}

func (s stubCatalogUseCase) SearchItems(ctx context.Context, filter model.SearchItemsFilter) ([]model.Item, error) {
	return nil, errors.New("not implemented")
}

func (s stubCatalogUseCase) GetPriceHistory(ctx context.Context, itemID string, filter model.PriceHistoryFilter) ([]model.PriceHistoryEntry, error) {
	return s.getPriceHistoryFn(ctx, itemID, filter)
}

func (s stubCatalogUseCase) GetStats(ctx context.Context) (*model.CatalogStatsResponse, error) {
	return nil, errors.New("not implemented")
}

func (s stubCatalogUseCase) GetLocalizationCoverage(ctx context.Context, game string) (*model.LocalizationCoverageResponse, error) {
	return nil, errors.New("not implemented")
}

func TestGetPriceHistory_NormalizesGameModeAndReturnsHistory(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	called := false
	h := New("catalog-service", stubCatalogUseCase{
		getPriceHistoryFn: func(ctx context.Context, itemID string, filter model.PriceHistoryFilter) ([]model.PriceHistoryEntry, error) {
			called = true
			if itemID != "item_1" {
				t.Fatalf("itemID = %q, want item_1", itemID)
			}
			if filter.GameMode != "pve" {
				t.Fatalf("game mode = %q, want pve", filter.GameMode)
			}
			if filter.Limit != 7 {
				t.Fatalf("limit = %d, want 7", filter.Limit)
			}

			return []model.PriceHistoryEntry{{
				ItemID:      itemID,
				Source:      "tarkov.dev",
				GameMode:    "pve",
				Value:       12345,
				Currency:    "RUB",
				CollectedOn: "2026-05-09",
				CollectedAt: time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC),
			}}, nil
		},
	}, nil)

	router := gin.New()
	router.GET("/items/:id/prices/history", h.GetPriceHistory)

	req := httptest.NewRequest(http.MethodGet, "/items/item_1/prices/history?game_mode=%20pve%20&limit=7", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if !called {
		t.Fatal("GetPriceHistory was not called")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp model.PriceHistoryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.ItemID != "item_1" {
		t.Fatalf("response item_id = %q, want item_1", resp.ItemID)
	}
	if resp.GameMode != "pve" {
		t.Fatalf("response game_mode = %q, want pve", resp.GameMode)
	}
	if len(resp.History) != 1 {
		t.Fatalf("history len = %d, want 1", len(resp.History))
	}
}

func TestGetPriceHistory_InvalidLimitReturnsBadRequest(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	h := New("catalog-service", stubCatalogUseCase{
		getPriceHistoryFn: func(ctx context.Context, itemID string, filter model.PriceHistoryFilter) ([]model.PriceHistoryEntry, error) {
			t.Fatal("GetPriceHistory must not be called")
			return nil, nil
		},
	}, nil)

	router := gin.New()
	router.GET("/items/:id/prices/history", h.GetPriceHistory)

	req := httptest.NewRequest(http.MethodGet, "/items/item_1/prices/history?limit=bad", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestGetPriceHistory_ItemNotFoundReturnsNotFound(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	h := New("catalog-service", stubCatalogUseCase{
		getPriceHistoryFn: func(ctx context.Context, itemID string, filter model.PriceHistoryFilter) ([]model.PriceHistoryEntry, error) {
			return nil, repository.ErrItemNotFound
		},
	}, nil)

	router := gin.New()
	router.GET("/items/:id/prices/history", h.GetPriceHistory)

	req := httptest.NewRequest(http.MethodGet, "/items/item_missing/prices/history", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestGetPriceHistory_InvalidFilterReturnsBadRequest(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	h := New("catalog-service", stubCatalogUseCase{
		getPriceHistoryFn: func(ctx context.Context, itemID string, filter model.PriceHistoryFilter) ([]model.PriceHistoryEntry, error) {
			return nil, service.ErrInvalidInput
		},
	}, nil)

	router := gin.New()
	router.GET("/items/:id/prices/history", h.GetPriceHistory)

	req := httptest.NewRequest(http.MethodGet, "/items/item_1/prices/history?limit=-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
