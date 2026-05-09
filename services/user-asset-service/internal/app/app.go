package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog"
	authclient "gtrade/services/user-asset-service/internal/client/auth"
	"gtrade/services/user-asset-service/internal/client/catalog"
	notificationclient "gtrade/services/user-asset-service/internal/client/notification"
	"gtrade/services/user-asset-service/internal/config"
	"gtrade/services/user-asset-service/internal/handler"
	httpserver "gtrade/services/user-asset-service/internal/http"
	"gtrade/services/user-asset-service/internal/repository"
	"gtrade/services/user-asset-service/internal/scheduler"
	"gtrade/services/user-asset-service/internal/service"
)

func Run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required for user-asset-service")
	}

	logger := zerolog.New(os.Stdout).With().Timestamp().Str("service", cfg.ServiceName).Logger()

	pool, err := repository.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()

	repo := repository.NewUserAssetRepository(pool)
	catalogClient := catalog.New(cfg.CatalogURL)
	userAssetService := service.NewUserAssetService(repo, catalogClient)
	authClient := authclient.New(cfg.AuthServiceURL, cfg.InternalAPIToken)
	notificationClient := notificationclient.New(cfg.NotificationServiceURL)
	priceAlertService := service.NewPriceAlertService(repo, catalogClient, authClient, notificationClient)
	h := handler.New(cfg.ServiceName, userAssetService)
	r := httpserver.NewRouter(logger, h)

	alertInterval, err := time.ParseDuration(cfg.PriceAlertCheckInterval)
	if err != nil {
		return fmt.Errorf("parse PRICE_ALERT_CHECK_INTERVAL: %w", err)
	}
	priceAlertScheduler := scheduler.NewPriceAlertScheduler(logger, priceAlertService, alertInterval)
	priceAlertScheduler.Start(ctx)

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
