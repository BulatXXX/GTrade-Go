package marketplace

import "gtrade/services/marketplace-integration-service/internal/model"

type TarkovClient struct{}

func NewTarkovClient() *TarkovClient { return &TarkovClient{} }

func (c *TarkovClient) SearchItems(_ string) ([]model.ItemDTO, error) { return []model.ItemDTO{}, nil }
func (c *TarkovClient) GetItemByID(id string) (*model.ItemDTO, error) {
	return &model.ItemDTO{ID: id, Game: "tarkov", Name: "stub", Currency: "rub"}, nil
}
func (c *TarkovClient) GetTopPrice(id string) (*model.PriceDTO, error) {
	return &model.PriceDTO{ItemID: id, Source: "tarkov", Value: 0, Currency: "rub"}, nil
}
