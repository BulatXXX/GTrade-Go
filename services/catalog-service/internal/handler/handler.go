package handler

import (
	"context"

	"gtrade/services/catalog-service/internal/model"
)

type CatalogUseCase interface {
	CreateItem(ctx context.Context, input model.CreateItemInput) (*model.Item, error)
	UpdateItem(ctx context.Context, id string, input model.UpdateItemInput) (*model.Item, error)
	DeleteItem(ctx context.Context, id string) error
	GetItemByID(ctx context.Context, id string) (*model.Item, error)
	ListItems(ctx context.Context, filter model.ListItemsFilter) ([]model.Item, error)
	SearchItems(ctx context.Context, filter model.SearchItemsFilter) ([]model.Item, error)
}

type Handler struct {
	serviceName    string
	catalogService CatalogUseCase
}

func New(serviceName string, catalogService CatalogUseCase) *Handler {
	return &Handler{serviceName: serviceName, catalogService: catalogService}
}
