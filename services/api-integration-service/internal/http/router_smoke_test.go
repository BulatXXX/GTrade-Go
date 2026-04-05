package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"gtrade/services/api-integration-service/internal/handler"
	"gtrade/services/api-integration-service/internal/model"
	"gtrade/services/api-integration-service/internal/service"
)

type stubIntegrationService struct {
	searchFn     func(ctx context.Context, query model.SearchItemsQuery) ([]model.Item, error)
	itemFn       func(ctx context.Context, query model.GetItemQuery) (*model.Item, error)
	priceFn      func(ctx context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error)
	syncItemFn   func(ctx context.Context, query model.SyncItemQuery) (*model.SyncedCatalogItem, error)
	syncSearchFn func(ctx context.Context, query model.SyncSearchQuery) ([]model.SyncedCatalogItem, error)
}

func (s stubIntegrationService) SearchItems(ctx context.Context, query model.SearchItemsQuery) ([]model.Item, error) {
	return s.searchFn(ctx, query)
}

func (s stubIntegrationService) GetItem(ctx context.Context, query model.GetItemQuery) (*model.Item, error) {
	return s.itemFn(ctx, query)
}

func (s stubIntegrationService) GetPricing(ctx context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error) {
	return s.priceFn(ctx, query)
}

func (s stubIntegrationService) SyncItemToCatalog(ctx context.Context, query model.SyncItemQuery) (*model.SyncedCatalogItem, error) {
	return s.syncItemFn(ctx, query)
}

func (s stubIntegrationService) SyncSearchToCatalog(ctx context.Context, query model.SyncSearchQuery) ([]model.SyncedCatalogItem, error) {
	return s.syncSearchFn(ctx, query)
}

func TestRouterSmoke(t *testing.T) {
	t.Parallel()

	router := NewRouter(zerolog.Nop(), handler.New("api-integration-service", stubIntegrationService{
		searchFn: func(ctx context.Context, query model.SearchItemsQuery) ([]model.Item, error) {
			if query.Game == "" || query.Query == "" {
				return nil, service.ErrInvalidInput
			}
			return []model.Item{{ID: "5448", Game: "tarkov", Name: "Makarov"}}, nil
		},
		itemFn: func(ctx context.Context, query model.GetItemQuery) (*model.Item, error) {
			if query.Game == "" {
				return nil, service.ErrInvalidInput
			}
			return &model.Item{ID: query.ID, Game: query.Game, GameMode: query.GameMode, Name: "stub"}, nil
		},
		priceFn: func(ctx context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error) {
			if query.Game == "" {
				return nil, service.ErrInvalidInput
			}
			if query.Game == "tarkov" && query.GameMode == "" {
				return nil, service.ErrInvalidInput
			}
			current := 10120.0
			return &model.PriceSnapshot{
				ItemID:     query.ID,
				Game:       query.Game,
				GameMode:   query.GameMode,
				Source:     "tarkov-dev",
				Currency:   "RUB",
				MarketKind: "aggregated_market",
				FetchedAt:  time.Unix(0, 0).UTC(),
				Pricing:    model.Pricing{Current: &current, Avg24h: &current},
			}, nil
		},
		syncItemFn: func(ctx context.Context, query model.SyncItemQuery) (*model.SyncedCatalogItem, error) {
			return &model.SyncedCatalogItem{ID: "item_1", Game: query.Game, ExternalID: query.ID, Name: "stub"}, nil
		},
		syncSearchFn: func(ctx context.Context, query model.SyncSearchQuery) ([]model.SyncedCatalogItem, error) {
			return []model.SyncedCatalogItem{{ID: "item_1", Game: query.Game, ExternalID: "5448", Name: "stub"}}, nil
		},
	}), "test-internal-token")

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantField  string
	}{
		{name: "health", method: http.MethodGet, path: "/health", wantStatus: http.StatusOK, wantField: "status"},
		{name: "search", method: http.MethodGet, path: "/search?game=tarkov&q=ak", wantStatus: http.StatusOK, wantField: "items"},
		{name: "get item", method: http.MethodGet, path: "/items/5448?game=tarkov", wantStatus: http.StatusOK, wantField: "item"},
		{name: "get prices", method: http.MethodGet, path: "/items/5448/prices?game=tarkov&game_mode=pve", wantStatus: http.StatusOK, wantField: "price"},
		{name: "get top price", method: http.MethodGet, path: "/items/5448/top-price?game=tarkov&game_mode=pve", wantStatus: http.StatusOK, wantField: "value"},
		{name: "search invalid", method: http.MethodGet, path: "/search?q=ak", wantStatus: http.StatusBadRequest, wantField: "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			var got map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if _, ok := got[tt.wantField]; !ok {
				t.Fatalf("missing field %q in %v", tt.wantField, got)
			}
		})
	}
}

func TestRouterSmoke_SyncEndpoints(t *testing.T) {
	t.Parallel()

	router := NewRouter(zerolog.Nop(), handler.New("api-integration-service", stubIntegrationService{
		searchFn: func(ctx context.Context, query model.SearchItemsQuery) ([]model.Item, error) { return nil, nil },
		itemFn:   func(ctx context.Context, query model.GetItemQuery) (*model.Item, error) { return nil, nil },
		priceFn:  func(ctx context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error) { return nil, nil },
		syncItemFn: func(ctx context.Context, query model.SyncItemQuery) (*model.SyncedCatalogItem, error) {
			if query.Game == "" || query.ID == "" {
				return nil, service.ErrInvalidInput
			}
			return &model.SyncedCatalogItem{ID: "item_1", Game: query.Game, ExternalID: query.ID, Name: "Frost Prime Set"}, nil
		},
		syncSearchFn: func(ctx context.Context, query model.SyncSearchQuery) ([]model.SyncedCatalogItem, error) {
			if query.Game == "" || query.Query == "" {
				return nil, service.ErrInvalidInput
			}
			return []model.SyncedCatalogItem{{ID: "item_1", Game: query.Game, ExternalID: "frost_prime_set", Name: "Frost Prime Set"}}, nil
		},
	}), "test-internal-token")

	tests := []struct {
		name       string
		path       string
		body       string
		wantStatus int
		wantField  string
	}{
		{name: "sync item", path: "/internal/sync/item", body: `{"game":"warframe","id":"frost_prime_set"}`, wantStatus: http.StatusOK, wantField: "item"},
		{name: "sync search", path: "/internal/sync/search", body: `{"game":"warframe","q":"frost","limit":1}`, wantStatus: http.StatusOK, wantField: "items"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Internal-Token", "test-internal-token")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			var got map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if _, ok := got[tt.wantField]; !ok {
				t.Fatalf("missing field %q in %v", tt.wantField, got)
			}
		})
	}
}

func TestRouterSmoke_SyncEndpointsRequireInternalToken(t *testing.T) {
	t.Parallel()

	router := NewRouter(zerolog.Nop(), handler.New("api-integration-service", stubIntegrationService{
		searchFn: func(ctx context.Context, query model.SearchItemsQuery) ([]model.Item, error) { return nil, nil },
		itemFn:   func(ctx context.Context, query model.GetItemQuery) (*model.Item, error) { return nil, nil },
		priceFn:  func(ctx context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error) { return nil, nil },
		syncItemFn: func(ctx context.Context, query model.SyncItemQuery) (*model.SyncedCatalogItem, error) {
			return nil, nil
		},
		syncSearchFn: func(ctx context.Context, query model.SyncSearchQuery) ([]model.SyncedCatalogItem, error) {
			return nil, nil
		},
	}), "test-internal-token")

	req := httptest.NewRequest(http.MethodPost, "/internal/sync/item", strings.NewReader(`{"game":"warframe","id":"frost_prime_set"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestRouterSmoke_SyncEndpointsFailClosedWhenInternalAuthIsUnconfigured(t *testing.T) {
	t.Parallel()

	router := NewRouter(zerolog.Nop(), handler.New("api-integration-service", stubIntegrationService{
		searchFn: func(ctx context.Context, query model.SearchItemsQuery) ([]model.Item, error) { return nil, nil },
		itemFn:   func(ctx context.Context, query model.GetItemQuery) (*model.Item, error) { return nil, nil },
		priceFn:  func(ctx context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error) { return nil, nil },
		syncItemFn: func(ctx context.Context, query model.SyncItemQuery) (*model.SyncedCatalogItem, error) {
			return nil, nil
		},
		syncSearchFn: func(ctx context.Context, query model.SyncSearchQuery) ([]model.SyncedCatalogItem, error) {
			return nil, nil
		},
	}), "")

	req := httptest.NewRequest(http.MethodPost, "/internal/sync/item", strings.NewReader(`{"game":"warframe","id":"frost_prime_set"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", "whatever")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}

func TestRouterSmoke_UpstreamFailureMapsToBadGateway(t *testing.T) {
	t.Parallel()

	router := NewRouter(zerolog.Nop(), handler.New("api-integration-service", stubIntegrationService{
		searchFn: func(ctx context.Context, query model.SearchItemsQuery) ([]model.Item, error) { return nil, nil },
		itemFn:   func(ctx context.Context, query model.GetItemQuery) (*model.Item, error) { return nil, nil },
		priceFn: func(ctx context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error) {
			return nil, service.ErrUpstreamFailed
		},
	}), "test-internal-token")

	req := httptest.NewRequest(http.MethodGet, "/items/5448/prices?game=tarkov", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadGateway, rec.Body.String())
	}

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !errors.Is(service.ErrUpstreamFailed, service.ErrUpstreamFailed) {
		t.Fatal("sentinel error must be stable")
	}
}
