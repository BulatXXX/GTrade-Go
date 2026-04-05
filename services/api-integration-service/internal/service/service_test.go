package service

import (
	"context"
	"errors"
	"testing"
	"time"

	catalogclient "gtrade/services/api-integration-service/internal/client/catalog"
	"gtrade/services/api-integration-service/internal/model"
	"gtrade/services/api-integration-service/internal/service/marketplace"
)

type stubProvider struct {
	game      string
	searchFn  func(ctx context.Context, query model.SearchItemsQuery) ([]model.Item, error)
	getItemFn func(ctx context.Context, query model.GetItemQuery) (*model.Item, error)
	priceFn   func(ctx context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error)
}

type stubCatalogWriter struct {
	upsertFn func(ctx context.Context, input catalogclient.UpsertItemRequest) (*catalogclient.Item, error)
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

func (s stubCatalogWriter) UpsertItem(ctx context.Context, input catalogclient.UpsertItemRequest) (*catalogclient.Item, error) {
	return s.upsertFn(ctx, input)
}

func TestServiceSearchItems_DelegatesToProvider(t *testing.T) {
	t.Parallel()

	svc := New(stubProvider{
		game: "tarkov",
		searchFn: func(ctx context.Context, query model.SearchItemsQuery) ([]model.Item, error) {
			if query.Game != "tarkov" || query.Query != "ak" || query.Limit != 20 || query.GameMode != "regular" {
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

func TestServiceGetPricing_DefaultsTarkovGameModeToRegular(t *testing.T) {
	t.Parallel()

	svc := New(stubProvider{
		game:      "tarkov",
		searchFn:  func(ctx context.Context, query model.SearchItemsQuery) ([]model.Item, error) { return nil, nil },
		getItemFn: func(ctx context.Context, query model.GetItemQuery) (*model.Item, error) { return nil, nil },
		priceFn: func(ctx context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error) {
			if query.GameMode != "regular" {
				t.Fatalf("game_mode = %q, want regular", query.GameMode)
			}
			return &model.PriceSnapshot{ItemID: query.ID, Game: query.Game, GameMode: query.GameMode}, nil
		},
	})

	_, err := svc.GetPricing(context.Background(), model.GetPricingQuery{
		Game: "tarkov",
		ID:   "5448bd6b4bdc2dfc2f8b4569",
	})
	if err != nil {
		t.Fatalf("GetPricing: %v", err)
	}
}

func TestServiceSyncItemToCatalog_UpsertsMappedItem(t *testing.T) {
	t.Parallel()

	svc := NewWithCatalog(stubCatalogWriter{
		upsertFn: func(ctx context.Context, input catalogclient.UpsertItemRequest) (*catalogclient.Item, error) {
			if input.Game != "warframe" || input.Source != "warframe-market" || input.ExternalID != "frost_prime_set" {
				t.Fatalf("unexpected upsert input: %#v", input)
			}
			if input.Slug != "frost_prime_set" || input.Name != "Frost Prime Set" {
				t.Fatalf("unexpected mapped fields: %#v", input)
			}
			return &catalogclient.Item{
				ID:         "item_1",
				Game:       input.Game,
				Source:     input.Source,
				ExternalID: input.ExternalID,
				Slug:       input.Slug,
				Name:       input.Name,
				UpdatedAt:  time.Unix(0, 0).UTC(),
			}, nil
		},
	}, stubProvider{
		game:     "warframe",
		searchFn: func(ctx context.Context, query model.SearchItemsQuery) ([]model.Item, error) { return nil, nil },
		getItemFn: func(ctx context.Context, query model.GetItemQuery) (*model.Item, error) {
			return &model.Item{
				ID:          "frost_prime_set",
				Game:        "warframe",
				Source:      "warframe-market",
				Slug:        "frost_prime_set",
				Name:        "Frost Prime Set",
				Description: "Prime set",
				ImageURL:    "https://example.com/frost.png",
			}, nil
		},
		priceFn: func(ctx context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error) { return nil, nil },
	})

	item, err := svc.SyncItemToCatalog(context.Background(), model.SyncItemQuery{
		Game: "warframe",
		ID:   "frost_prime_set",
	})
	if err != nil {
		t.Fatalf("SyncItemToCatalog: %v", err)
	}
	if item.ID != "item_1" || item.ExternalID != "frost_prime_set" {
		t.Fatalf("unexpected synced item: %#v", item)
	}
}

func TestServiceSyncSearchToCatalog_DefaultsTarkovGameModeToRegular(t *testing.T) {
	t.Parallel()

	svc := NewWithCatalog(stubCatalogWriter{
		upsertFn: func(ctx context.Context, input catalogclient.UpsertItemRequest) (*catalogclient.Item, error) {
			return &catalogclient.Item{
				ID:         "item_5448",
				Game:       input.Game,
				Source:     input.Source,
				ExternalID: input.ExternalID,
				Slug:       input.Slug,
				Name:       input.Name,
			}, nil
		},
	}, stubProvider{
		game: "tarkov",
		searchFn: func(ctx context.Context, query model.SearchItemsQuery) ([]model.Item, error) {
			if query.GameMode != "regular" {
				t.Fatalf("game_mode = %q, want regular", query.GameMode)
			}
			return []model.Item{{
				ID:       "5448",
				Game:     "tarkov",
				GameMode: query.GameMode,
				Source:   "tarkov-dev",
				Name:     "Makarov",
				Slug:     "5448",
			}}, nil
		},
		getItemFn: func(ctx context.Context, query model.GetItemQuery) (*model.Item, error) { return nil, nil },
		priceFn:   func(ctx context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error) { return nil, nil },
	})

	items, err := svc.SyncSearchToCatalog(context.Background(), model.SyncSearchQuery{
		Game:  "tarkov",
		Query: "mak",
		Limit: 1,
	})
	if err != nil {
		t.Fatalf("SyncSearchToCatalog: %v", err)
	}
	if len(items) != 1 || items[0].ExternalID != "5448" {
		t.Fatalf("unexpected synced items: %#v", items)
	}
}
