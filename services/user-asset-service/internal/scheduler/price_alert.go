package scheduler

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

type PriceAlertRunner interface {
	RunCycle(ctx context.Context, now time.Time) error
}

type PriceAlertScheduler struct {
	logger   zerolog.Logger
	runner   PriceAlertRunner
	interval time.Duration
}

func NewPriceAlertScheduler(logger zerolog.Logger, runner PriceAlertRunner, interval time.Duration) *PriceAlertScheduler {
	return &PriceAlertScheduler{
		logger:   logger,
		runner:   runner,
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
	runCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	if err := s.runner.RunCycle(runCtx, now); err != nil {
		s.logger.Error().Err(err).Msg("price alert scheduler cycle failed")
		return
	}

	s.logger.Info().Str("run_at", now.UTC().Format(time.RFC3339)).Msg("price alert scheduler cycle completed")
}
