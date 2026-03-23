package marketplace

import "gtrade/services/marketplace-integration-service/internal/model"

type WarframeClient struct{}

func NewWarframeClient() *WarframeClient { return &WarframeClient{} }

func (c *WarframeClient) SearchItems(_ string) ([]model.ItemDTO, error) {
	return []model.ItemDTO{}, nil
}
func (c *WarframeClient) GetItemByID(id string) (*model.ItemDTO, error) {
	return &model.ItemDTO{ID: id, Game: "warframe", Name: "stub", Currency: "plat"}, nil
}
func (c *WarframeClient) GetTopPrice(id string) (*model.PriceDTO, error) {
	return &model.PriceDTO{ItemID: id, Source: "warframe", Value: 0, Currency: "plat"}, nil
}
