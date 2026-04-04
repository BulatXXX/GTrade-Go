package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog"
	"gtrade/services/notification-service/internal/config"
	"gtrade/services/notification-service/internal/handler"
	httpserver "gtrade/services/notification-service/internal/http"
	"gtrade/services/notification-service/internal/repository"
	"gtrade/services/notification-service/internal/service"
	"gtrade/services/notification-service/internal/service/provider"
)

func Run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := zerolog.New(os.Stdout).With().Timestamp().Str("service", cfg.ServiceName).Logger()

	if cfg.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	pool, err := repository.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()

	repo := repository.NewNotificationRepository(pool)
	emailProvider, err := newEmailProvider(cfg, logger)
	if err != nil {
		return fmt.Errorf("configure email provider: %w", err)
	}
	emailService := service.NewEmailService(repo, emailProvider, cfg.ResendFromEmail)

	h := handler.New(cfg.ServiceName, emailService)
	r := httpserver.NewRouter(logger, h)

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

func newEmailProvider(cfg config.Config, logger zerolog.Logger) (provider.EmailProvider, error) {
	switch cfg.EmailProvider {
	case "mock":
		return provider.NewMockProvider(logger), nil
	case "resend":
		return provider.NewResendProvider(cfg.ResendAPIKey), nil
	default:
		return nil, fmt.Errorf("unsupported email provider: %s", cfg.EmailProvider)
	}
}
