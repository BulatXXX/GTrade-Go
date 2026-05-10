package scheduler

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

const PriceAlertJobName = "price_alert_dispatch"

type PriceAlertRunner interface {
	RunCycle(ctx context.Context, now time.Time) error
}

type SchedulerStateStore interface {
	AcquireSchedulerLock(ctx context.Context, lockKey int64) (bool, func(), error)
	MarkSchedulerStarted(ctx context.Context, jobName string, startedAt time.Time) error
	MarkSchedulerFinished(ctx context.Context, jobName string, finishedAt time.Time, runErr error, processed, total int) error
}

type PriceAlertScheduler struct {
	logger   zerolog.Logger
	runner   PriceAlertRunner
	store    SchedulerStateStore
	lockKey  int64
	interval time.Duration
}

func NewPriceAlertScheduler(logger zerolog.Logger, runner PriceAlertRunner, store SchedulerStateStore, lockKey int64, interval time.Duration) *PriceAlertScheduler {
	return &PriceAlertScheduler{
		logger:   logger.With().Str("component", "price_alert_scheduler").Logger(),
		runner:   runner,
		store:    store,
		lockKey:  lockKey,
		interval: interval,
	}
}

func (s *PriceAlertScheduler) Start(ctx context.Context) {
	if s == nil || s.runner == nil || s.interval <= 0 {
		return
	}

	go func() {
		s.runOnce(ctx)

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				s.runAt(ctx, now)
			}
		}
	}()
}

func (s *PriceAlertScheduler) runOnce(ctx context.Context) {
	s.runAt(ctx, time.Now().UTC())
}

func (s *PriceAlertScheduler) runAt(ctx context.Context, now time.Time) {
	if s.store != nil {
		acquired, release, err := s.store.AcquireSchedulerLock(ctx, s.lockKey)
		if err != nil {
			s.logger.Error().Err(err).Msg("acquire price alert lock failed")
			return
		}
		if !acquired {
			s.logger.Info().Msg("price alert cycle skipped: lock busy")
			return
		}
		defer release()

		if err := s.store.MarkSchedulerStarted(ctx, PriceAlertJobName, now); err != nil {
			s.logger.Warn().Err(err).Msg("mark scheduler started failed")
		}
	}

	runCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	runErr := s.runner.RunCycle(runCtx, now)
	finishedAt := time.Now().UTC()

	if s.store != nil {
		if err := s.store.MarkSchedulerFinished(ctx, PriceAlertJobName, finishedAt, runErr, 0, 0); err != nil {
			s.logger.Warn().Err(err).Msg("mark scheduler finished failed")
		}
	}

	if runErr != nil {
		s.logger.Error().Err(runErr).Msg("price alert scheduler cycle failed")
		return
	}

	s.logger.Info().Str("run_at", now.UTC().Format(time.RFC3339)).Msg("price alert scheduler cycle completed")
}
