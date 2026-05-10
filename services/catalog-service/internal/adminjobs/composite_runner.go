package adminjobs

import (
	"context"
	"fmt"

	"gtrade/services/catalog-service/internal/model"
	"gtrade/services/catalog-service/internal/repository"
)

type SchedulerStateLister interface {
	ListSchedulerStates(ctx context.Context) ([]repository.SchedulerState, error)
}

type CompositeRunner struct {
	priceHistory   *PriceHistorySyncRunner
	catalogImport  *CatalogImportRunner
	schedulerState SchedulerStateLister
}

func NewCompositeRunner(priceHistory *PriceHistorySyncRunner, catalogImport *CatalogImportRunner, schedulerState SchedulerStateLister) *CompositeRunner {
	return &CompositeRunner{
		priceHistory:   priceHistory,
		catalogImport:  catalogImport,
		schedulerState: schedulerState,
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

func (r *CompositeRunner) ListSchedulerStates(ctx context.Context) (*model.SchedulerStateResponse, error) {
	if r.schedulerState == nil {
		return &model.SchedulerStateResponse{Items: []model.SchedulerStateItem{}}, nil
	}
	states, err := r.schedulerState.ListSchedulerStates(ctx)
	if err != nil {
		return nil, fmt.Errorf("list scheduler states: %w", err)
	}
	out := make([]model.SchedulerStateItem, 0, len(states))
	for _, st := range states {
		var startedAt, finishedAt *string
		if st.LastStartedAt != nil {
			s := st.LastStartedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
			startedAt = &s
		}
		if st.LastFinishedAt != nil {
			s := st.LastFinishedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
			finishedAt = &s
		}
		out = append(out, model.SchedulerStateItem{
			JobName:        st.JobName,
			Status:         st.Status,
			LastStartedAt:  startedAt,
			LastFinishedAt: finishedAt,
			LastError:      st.LastError,
			LastProcessed:  st.LastProcessed,
			LastTotal:      st.LastTotal,
			RunsTotal:      st.RunsTotal,
			UpdatedAt:      st.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return &model.SchedulerStateResponse{Items: out}, nil
}
