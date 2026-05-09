package adminjobs

import (
	"context"

	"gtrade/services/catalog-service/internal/scheduler"
)

type PriceHistorySyncRunner struct {
	manager   *Manager
	collector *scheduler.PriceHistoryCollector
}

func NewPriceHistorySyncRunner(manager *Manager, collector *scheduler.PriceHistoryCollector) *PriceHistorySyncRunner {
	return &PriceHistorySyncRunner{
		manager:   manager,
		collector: collector,
	}
}

func (r *PriceHistorySyncRunner) StartPriceHistorySync(ctx context.Context) *Job {
	return r.manager.Start(ctx, "price-history-sync", func(ctx context.Context, job *Job) error {
		observer := &progressObserver{manager: r.manager, jobID: job.ID}
		r.collector.RunOnce(ctx, observer)
		return nil
	})
}

func (r *PriceHistorySyncRunner) GetJob(id string) *Job {
	return r.manager.Get(id)
}

func (r *PriceHistorySyncRunner) ListJobs() []*Job {
	return r.manager.List()
}

type progressObserver struct {
	manager   *Manager
	jobID     string
	processed int
	total     int
}

func (o *progressObserver) OnStart(total int) {
	o.total = total
	o.manager.UpdateProgress(o.jobID, o.processed, o.total)
}

func (o *progressObserver) OnItemProcessed() {
	o.processed++
	o.manager.UpdateProgress(o.jobID, o.processed, o.total)
}

func (o *progressObserver) OnFinish() {
	o.manager.UpdateProgress(o.jobID, o.total, o.total)
}
