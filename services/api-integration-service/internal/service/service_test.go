package service

import (
	"context"
	"errors"
	"testing"

	"gtrade/services/api-integration-service/internal/model"
	"gtrade/services/api-integration-service/internal/service/marketplace"
)

type stubProvider struct {
	game      string
	searchFn  func(ctx context.Context, query model.SearchItemsQuery) ([]model.Item, error)
	getItemFn func(ctx context.Context, query model.GetItemQuery) (*model.Item, error)
	priceFn   func(ctx context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error)
}

func (s stubProvider) Game() string { return s.game }

func (s stubProvider) SearchItems(ctx context.Context, query model.SearchItemsQuery) ([]model.Item, error) {
	return s.searchFn(ctx, query)
}

func (s stubProvider) GetItem(ctx context.Context, query model.GetItemQuery) (*model.Item, error) {
	return s.getItemFn(ctx, query)
}

func (s stubProvider) GetPricing(ctx context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error) {
	return s.priceFn(ctx, query)
}

func TestServiceSearchItems_DelegatesToProvider(t *testing.T) {
	t.Parallel()

	svc := New(stubProvider{
		game: "tarkov",
		searchFn: func(ctx context.Context, query model.SearchItemsQuery) ([]model.Item, error) {
			if query.Game != "tarkov" || query.Query != "ak" || query.Limit != 20 {
				t.Fatalf("unexpected query: %#v", query)
			}
			return []model.Item{{ID: "1", Game: "tarkov", Name: "AK"}}, nil
		},
		getItemFn: func(ctx context.Context, query model.GetItemQuery) (*model.Item, error) { return nil, nil },
		priceFn:   func(ctx context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error) { return nil, nil },
	})

	items, err := svc.SearchItems(context.Background(), model.SearchItemsQuery{
		Game:  "tarkov",
		Query: "ak",
	})
	if err != nil {
		t.Fatalf("SearchItems: %v", err)
	}
	if len(items) != 1 || items[0].ID != "1" {
		t.Fatalf("unexpected items: %#v", items)
	}
}

func TestServiceSearchItems_UnsupportedGame(t *testing.T) {
	t.Parallel()

	svc := New()
	_, err := svc.SearchItems(context.Background(), model.SearchItemsQuery{
		Game:  "unknown",
		Query: "ak",
	})
	if !errors.Is(err, ErrUnsupportedGame) {
		t.Fatalf("error = %v, want %v", err, ErrUnsupportedGame)
	}
}

func TestServiceGetItem_MapsNotFound(t *testing.T) {
	t.Parallel()

	svc := New(stubProvider{
		game:     "warframe",
		searchFn: func(ctx context.Context, query model.SearchItemsQuery) ([]model.Item, error) { return nil, nil },
		getItemFn: func(ctx context.Context, query model.GetItemQuery) (*model.Item, error) {
			return nil, marketplace.ErrNotFound
		},
		priceFn: func(ctx context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error) { return nil, nil },
	})

	_, err := svc.GetItem(context.Background(), model.GetItemQuery{Game: "warframe", ID: "frost"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want %v", err, ErrNotFound)
	}
}

func TestServiceGetPricing_ValidatesInput(t *testing.T) {
	t.Parallel()

	svc := New()
	_, err := svc.GetPricing(context.Background(), model.GetPricingQuery{})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidInput)
	}
}
