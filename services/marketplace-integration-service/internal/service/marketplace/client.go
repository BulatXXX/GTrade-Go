package marketplace

import "gtrade/services/marketplace-integration-service/internal/model"

type MarketplaceClient interface {
	SearchItems(query string) ([]model.ItemDTO, error)
	GetItemByID(id string) (*model.ItemDTO, error)
	GetTopPrice(id string) (*model.PriceDTO, error)
}
