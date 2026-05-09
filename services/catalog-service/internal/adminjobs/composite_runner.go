package adminjobs

import (
	"context"

	"gtrade/services/catalog-service/internal/model"
)

type CompositeRunner struct {
	priceHistory  *PriceHistorySyncRunner
	catalogImport *CatalogImportRunner
}

func NewCompositeRunner(priceHistory *PriceHistorySyncRunner, catalogImport *CatalogImportRunner) *CompositeRunner {
	return &CompositeRunner{
		priceHistory:  priceHistory,
		catalogImport: catalogImport,
	}
}

func (r *CompositeRunner) StartPriceHistorySync(ctx context.Context) *Job {
	return r.priceHistory.StartPriceHistorySync(ctx)
}

func (r *CompositeRunner) StartCatalogImport(ctx context.Context, req model.AdminCatalogImportRequest) (*Job, error) {
	return r.catalogImport.StartCatalogImport(ctx, req)
}

func (r *CompositeRunner) GetJob(id string) *Job {
	return r.priceHistory.GetJob(id)
}

func (r *CompositeRunner) ListJobs() []*Job {
	return r.priceHistory.ListJobs()
}
