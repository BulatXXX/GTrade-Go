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

type CatalogService interface {
	ListActiveItemsForPriceSync(ctx context.Context, limit, offset int) ([]model.Item, error)
	UpsertPriceHistory(ctx context.Context, input model.UpsertPriceHistoryInput) error
}

type PriceHistoryCollector struct {
	logger   zerolog.Logger
	service  CatalogService
	client   *integration.Client
	interval time.Duration
}

func NewPriceHistoryCollector(
	logger zerolog.Logger,
	service CatalogService,
	client *integration.Client,
	interval time.Duration,
) *PriceHistoryCollector {
	return &PriceHistoryCollector{
		logger:   logger,
		service:  service,
		client:   client,
		interval: interval,
	}
}

func (c *PriceHistoryCollector) Start(ctx context.Context) {
	if c == nil || c.client == nil || c.service == nil || c.interval <= 0 {
		return
	}

	go func() {
		c.runOnce(ctx)

		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.runOnce(ctx)
			}
		}
	}()
}

func (c *PriceHistoryCollector) runOnce(ctx context.Context) {
	const batchSize = 100

	runCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

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
		}
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
