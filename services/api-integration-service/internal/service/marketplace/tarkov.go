package marketplace

import (
	"context"
	"time"

	"gtrade/services/api-integration-service/internal/model"
)

type TarkovClient struct{}

func NewTarkovClient() *TarkovClient { return &TarkovClient{} }

func (c *TarkovClient) Game() string { return "tarkov" }

func (c *TarkovClient) SearchItems(ctx context.Context, query model.SearchItemsQuery) ([]model.Item, error) {
	_ = ctx
	_ = query
	return []model.Item{}, nil
}

func (c *TarkovClient) GetItem(ctx context.Context, query model.GetItemQuery) (*model.Item, error) {
	_ = ctx
	return &model.Item{
		ID:       query.ID,
		Game:     "tarkov",
		Source:   "tarkov-dev",
		Name:     "stub",
		Currency: "RUB",
	}, nil
}

func (c *TarkovClient) GetPricing(ctx context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error) {
	_ = ctx
	value := 0.0
	return &model.PriceSnapshot{
		ItemID:     query.ID,
		Game:       "tarkov",
		Source:     "tarkov-dev",
		Currency:   "RUB",
		MarketKind: "aggregated_market",
		FetchedAt:  time.Now().UTC(),
		Pricing: model.Pricing{
			Current: &value,
			Avg24h:  &value,
		},
	}, nil
}
