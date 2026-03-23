package marketplace

import "gtrade/services/marketplace-integration-service/internal/model"

type EVEClient struct{}

func NewEVEClient() *EVEClient { return &EVEClient{} }

func (c *EVEClient) SearchItems(_ string) ([]model.ItemDTO, error) { return []model.ItemDTO{}, nil }
func (c *EVEClient) GetItemByID(id string) (*model.ItemDTO, error) {
	return &model.ItemDTO{ID: id, Game: "eve", Name: "stub", Currency: "isk"}, nil
}
func (c *EVEClient) GetTopPrice(id string) (*model.PriceDTO, error) {
	return &model.PriceDTO{ItemID: id, Source: "eve", Value: 0, Currency: "isk"}, nil
}
