package marketplace

import (
	"context"
	"time"

	"gtrade/services/api-integration-service/internal/model"
)

type EVEClient struct{}

func NewEVEClient() *EVEClient { return &EVEClient{} }

func (c *EVEClient) Game() string { return "eve" }

func (c *EVEClient) SearchItems(_ context.Context, _ model.SearchItemsQuery) ([]model.Item, error) {
	return []model.Item{}, nil
}

func (c *EVEClient) GetItem(_ context.Context, query model.GetItemQuery) (*model.Item, error) {
	return &model.Item{ID: query.ID, Game: "eve", Source: "esi", Name: "stub", Currency: "ISK"}, nil
}

func (c *EVEClient) GetPricing(_ context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error) {
	average := 0.0
	adjusted := 0.0
	return &model.PriceSnapshot{
		ItemID:     query.ID,
		Game:       "eve",
		Source:     "esi",
		Currency:   "ISK",
		MarketKind: "reference_market",
		FetchedAt:  time.Now().UTC(),
		Pricing: model.Pricing{
			Current:       &average,
			AdjustedPrice: &adjusted,
		},
	}, nil
}
