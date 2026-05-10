package handler

import (
	"context"

	"gtrade/services/catalog-service/internal/adminjobs"
	"gtrade/services/catalog-service/internal/model"
)

type CatalogUseCase interface {
	CreateItem(ctx context.Context, input model.CreateItemInput) (*model.Item, error)
	UpsertItem(ctx context.Context, input model.CreateItemInput) (*model.Item, error)
	UpdateItem(ctx context.Context, id string, input model.UpdateItemInput) (*model.Item, error)
	DeleteItem(ctx context.Context, id string) error
	GetItemByID(ctx context.Context, id string) (*model.Item, error)
	ListItems(ctx context.Context, filter model.ListItemsFilter) ([]model.Item, error)
	SearchItems(ctx context.Context, filter model.SearchItemsFilter) ([]model.Item, error)
	GetPriceHistory(ctx context.Context, itemID string, filter model.PriceHistoryFilter) ([]model.PriceHistoryEntry, error)
	GetStats(ctx context.Context) (*model.CatalogStatsResponse, error)
	GetLocalizationCoverage(ctx context.Context, game string) (*model.LocalizationCoverageResponse, error)
}

type AdminUseCase interface {
	StartPriceHistorySync(ctx context.Context) *adminjobs.Job
	StartCatalogImport(ctx context.Context, req model.AdminCatalogImportRequest) (*adminjobs.Job, error)
	GetJob(id string) *adminjobs.Job
	ListJobs() []*adminjobs.Job
	ListSchedulerStates(ctx context.Context) (*model.SchedulerStateResponse, error)
}

type Handler struct {
	serviceName    string
	catalogService CatalogUseCase
	adminService   AdminUseCase
}

func New(serviceName string, catalogService CatalogUseCase, adminService AdminUseCase) *Handler {
	return &Handler{serviceName: serviceName, catalogService: catalogService, adminService: adminService}
}
