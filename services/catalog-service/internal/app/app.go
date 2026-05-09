package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog"
	"gtrade/services/catalog-service/internal/client/integration"
	"gtrade/services/catalog-service/internal/config"
	"gtrade/services/catalog-service/internal/handler"
	httpserver "gtrade/services/catalog-service/internal/http"
	"gtrade/services/catalog-service/internal/repository"
	"gtrade/services/catalog-service/internal/scheduler"
	"gtrade/services/catalog-service/internal/service"
)

func Run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := zerolog.New(os.Stdout).With().Timestamp().Str("service", cfg.ServiceName).Logger()

	if cfg.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is empty")
	}

	pool, err := repository.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()

	repo := repository.NewCatalogRepository(pool)
	svc := service.New(repo)
	h := handler.New(cfg.ServiceName, svc)
	r := httpserver.NewRouter(logger, h)

	refreshInterval, err := time.ParseDuration(cfg.PriceHistoryRefreshInterval)
	if err != nil {
		return fmt.Errorf("parse PRICE_HISTORY_REFRESH_INTERVAL: %w", err)
	}

	integrationClient := integration.New(cfg.IntegrationServiceURL)
	priceCollector := scheduler.NewPriceHistoryCollector(logger, svc, integrationClient, refreshInterval)
	priceCollector.Start(ctx)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info().Int("port", cfg.Port).Msg("server starting")
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	}
}
