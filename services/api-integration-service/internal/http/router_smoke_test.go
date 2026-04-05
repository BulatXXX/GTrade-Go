package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"gtrade/services/api-integration-service/internal/handler"
	"gtrade/services/api-integration-service/internal/model"
	"gtrade/services/api-integration-service/internal/service"
)

type stubIntegrationService struct {
	searchFn func(ctx context.Context, query model.SearchItemsQuery) ([]model.Item, error)
	itemFn   func(ctx context.Context, query model.GetItemQuery) (*model.Item, error)
	priceFn  func(ctx context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error)
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
	}))

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

func TestRouterSmoke_UpstreamFailureMapsToBadGateway(t *testing.T) {
	t.Parallel()

	router := NewRouter(zerolog.Nop(), handler.New("api-integration-service", stubIntegrationService{
		searchFn: func(ctx context.Context, query model.SearchItemsQuery) ([]model.Item, error) { return nil, nil },
		itemFn:   func(ctx context.Context, query model.GetItemQuery) (*model.Item, error) { return nil, nil },
		priceFn: func(ctx context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error) {
			return nil, service.ErrUpstreamFailed
		},
	}))

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
