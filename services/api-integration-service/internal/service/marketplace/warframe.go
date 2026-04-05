package marketplace

import (
	"context"
	"time"

	"gtrade/services/api-integration-service/internal/model"
)

type WarframeClient struct{}

func NewWarframeClient() *WarframeClient { return &WarframeClient{} }

func (c *WarframeClient) Game() string { return "warframe" }

func (c *WarframeClient) SearchItems(_ context.Context, _ model.SearchItemsQuery) ([]model.Item, error) {
	return []model.Item{}, nil
}

func (c *WarframeClient) GetItem(_ context.Context, query model.GetItemQuery) (*model.Item, error) {
	return &model.Item{ID: query.ID, Game: "warframe", Source: "warframe-market", Name: "stub", Currency: "PLAT"}, nil
}

func (c *WarframeClient) GetPricing(_ context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error) {
	topSell := 0.0
	topBuy := 0.0
	spread := topSell - topBuy
	return &model.PriceSnapshot{
		ItemID:     query.ID,
		Game:       "warframe",
		Source:     "warframe-market",
		Currency:   "PLAT",
		MarketKind: "live_orders",
		FetchedAt:  time.Now().UTC(),
		Pricing: model.Pricing{
			Current: &topSell,
			TopSell: &topSell,
			TopBuy:  &topBuy,
			Spread:  &spread,
		},
	}, nil
}
