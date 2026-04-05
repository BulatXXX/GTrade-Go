package handler

import (
	"context"

	"gtrade/services/api-integration-service/internal/model"
)

type IntegrationUseCase interface {
	SearchItems(ctx context.Context, query model.SearchItemsQuery) ([]model.Item, error)
	GetItem(ctx context.Context, query model.GetItemQuery) (*model.Item, error)
	GetPricing(ctx context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error)
}

type Handler struct {
	serviceName string
	service     IntegrationUseCase
}

func New(serviceName string, service IntegrationUseCase) *Handler {
	return &Handler{serviceName: serviceName, service: service}
}
