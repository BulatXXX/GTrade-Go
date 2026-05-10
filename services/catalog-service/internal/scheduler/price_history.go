package scheduler

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"gtrade/services/catalog-service/internal/client/integration"
	"gtrade/services/catalog-service/internal/model"
	"gtrade/services/catalog-service/internal/service"
)

const PriceHistoryJobName = "price_history_sync"

type CatalogService interface {
	ListActiveItemsForPriceSync(ctx context.Context, limit, offset int) ([]model.Item, error)
	UpsertPriceHistory(ctx context.Context, input model.UpsertPriceHistoryInput) error
}

type SchedulerStateStore interface {
	AcquireSchedulerLock(ctx context.Context, lockKey int64) (bool, func(), error)
	MarkSchedulerStarted(ctx context.Context, jobName string, startedAt time.Time) error
	MarkSchedulerFinished(ctx context.Context, jobName string, finishedAt time.Time, runErr error, processed, total int) error
}

type PriceHistoryCollector struct {
	logger   zerolog.Logger
	service  CatalogService
	client   *integration.Client
	store    SchedulerStateStore
	lockKey  int64
	interval time.Duration
}

type ProgressObserver interface {
	OnStart(total int)
	OnItemProcessed()
	OnFinish()
}

func NewPriceHistoryCollector(
	logger zerolog.Logger,
	service CatalogService,
	client *integration.Client,
	store SchedulerStateStore,
	lockKey int64,
	interval time.Duration,
) *PriceHistoryCollector {
	return &PriceHistoryCollector{
		logger:   logger.With().Str("component", "price_history_collector").Logger(),
		service:  service,
		client:   client,
		store:    store,
		lockKey:  lockKey,
		interval: interval,
	}
}

func (c *PriceHistoryCollector) Start(ctx context.Context) {
	if c == nil || c.client == nil || c.service == nil || c.interval <= 0 {
		return
	}

	go func() {
		c.runWithLock(ctx, nil)

		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.runWithLock(ctx, nil)
			}
		}
	}()
}

// runWithLock guards RunOnce with a session-level advisory lock. If another
// instance (or the admin endpoint) is currently running the same job the call
// is skipped silently — the next tick or admin click will retry.
func (c *PriceHistoryCollector) runWithLock(ctx context.Context, observer ProgressObserver) {
	if c.store == nil {
		c.RunOnce(ctx, observer)
		return
	}
	acquired, release, err := c.store.AcquireSchedulerLock(ctx, c.lockKey)
	if err != nil {
		c.logger.Error().Err(err).Msg("acquire price history lock failed")
		return
	}
	if !acquired {
		c.logger.Info().Msg("price history sync skipped: lock busy")
		return
	}
	defer release()

	startedAt := time.Now().UTC()
	if err := c.store.MarkSchedulerStarted(ctx, PriceHistoryJobName, startedAt); err != nil {
		c.logger.Warn().Err(err).Msg("mark price history started failed")
	}

	tracker := &countingObserver{inner: observer}
	c.RunOnce(ctx, tracker)

	if err := c.store.MarkSchedulerFinished(ctx, PriceHistoryJobName, time.Now().UTC(), nil, tracker.processed, tracker.total); err != nil {
		c.logger.Warn().Err(err).Msg("mark price history finished failed")
	}
}

// RunWithExternalLock is invoked by the admin handler to share the same
// advisory lock as the background ticker. Returns false when the lock is
// already held — caller should respond with HTTP 409.
func (c *PriceHistoryCollector) RunWithExternalLock(ctx context.Context, observer ProgressObserver) (bool, error) {
	if c.store == nil {
		c.RunOnce(ctx, observer)
		return true, nil
	}
	acquired, release, err := c.store.AcquireSchedulerLock(ctx, c.lockKey)
	if err != nil {
		return false, err
	}
	if !acquired {
		return false, nil
	}
	defer release()

	startedAt := time.Now().UTC()
	if err := c.store.MarkSchedulerStarted(ctx, PriceHistoryJobName, startedAt); err != nil {
		c.logger.Warn().Err(err).Msg("mark price history started failed")
	}

	tracker := &countingObserver{inner: observer}
	c.RunOnce(ctx, tracker)

	if err := c.store.MarkSchedulerFinished(ctx, PriceHistoryJobName, time.Now().UTC(), nil, tracker.processed, tracker.total); err != nil {
		c.logger.Warn().Err(err).Msg("mark price history finished failed")
	}
	return true, nil
}

type countingObserver struct {
	inner     ProgressObserver
	processed int
	total     int
}

func (o *countingObserver) OnStart(total int) {
	o.total = total
	if o.inner != nil {
		o.inner.OnStart(total)
	}
}

func (o *countingObserver) OnItemProcessed() {
	o.processed++
	if o.inner != nil {
		o.inner.OnItemProcessed()
	}
}

func (o *countingObserver) OnFinish() {
	if o.inner != nil {
		o.inner.OnFinish()
	}
}

func (c *PriceHistoryCollector) RunOnce(ctx context.Context, observer ProgressObserver) {
	const batchSize = 100

	runCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	total := 0
	for offset := 0; ; offset += batchSize {
		items, err := c.service.ListActiveItemsForPriceSync(runCtx, batchSize, offset)
		if err != nil {
			c.logger.Error().Err(err).Msg("price history: list active items failed")
			return
		}
		if len(items) == 0 {
			break
		}
		total += len(items)
		if len(items) < batchSize {
			break
		}
	}
	if observer != nil {
		observer.OnStart(total)
	}

	var processed int
	for offset := 0; ; offset += batchSize {
		items, err := c.service.ListActiveItemsForPriceSync(runCtx, batchSize, offset)
		if err != nil {
			c.logger.Error().Err(err).Msg("price history: list active items failed")
			return
		}
		if len(items) == 0 {
			break
		}

		for _, item := range items {
			processed += c.collectItem(runCtx, item)
			if observer != nil {
				observer.OnItemProcessed()
			}
		}
	}

	if observer != nil {
		observer.OnFinish()
	}

	c.logger.Info().Int("processed_items", processed).Msg("price history: sync finished")
}

func (c *PriceHistoryCollector) collectItem(ctx context.Context, item model.Item) int {
	gameModes := []string{""}
	if strings.EqualFold(item.Game, "tarkov") {
		gameModes = []string{"regular", "pve"}
	}

	processed := 0
	for _, gameMode := range gameModes {
		price, err := c.client.GetTopPrice(ctx, item.ExternalID, item.Game, gameMode)
		if err != nil {
			if errors.Is(err, integration.ErrNotFound) {
				c.logger.Warn().
					Str("item_id", item.ID).
					Str("external_id", item.ExternalID).
					Str("game", item.Game).
					Str("game_mode", gameMode).
					Msg("price history: top price not found")
				continue
			}

			c.logger.Error().
				Err(err).
				Str("item_id", item.ID).
				Str("external_id", item.ExternalID).
				Str("game", item.Game).
				Str("game_mode", gameMode).
				Msg("price history: top price fetch failed")
			continue
		}

		if price.Value == nil || price.Currency == "" || price.FetchedAt.IsZero() {
			c.logger.Warn().
				Str("item_id", item.ID).
				Str("external_id", item.ExternalID).
				Str("game", item.Game).
				Str("game_mode", gameMode).
				Msg("price history: top price response incomplete")
			continue
		}

		if err := c.service.UpsertPriceHistory(ctx, model.UpsertPriceHistoryInput{
			ItemID:      item.ID,
			Source:      price.Source,
			GameMode:    price.GameMode,
			Value:       *price.Value,
			Currency:    price.Currency,
			CollectedOn: price.FetchedAt,
			CollectedAt: price.FetchedAt,
		}); err != nil {
			if errors.Is(err, service.ErrInvalidInput) {
				c.logger.Error().
					Err(err).
					Str("item_id", item.ID).
					Str("external_id", item.ExternalID).
					Str("game", item.Game).
					Str("game_mode", gameMode).
					Msg("price history: invalid price payload")
				continue
			}

			c.logger.Error().
				Err(err).
				Str("item_id", item.ID).
				Str("external_id", item.ExternalID).
				Str("game", item.Game).
				Str("game_mode", gameMode).
				Msg("price history: store failed")
			continue
		}

		processed++
	}

	return processed
}
