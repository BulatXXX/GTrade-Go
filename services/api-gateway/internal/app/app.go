package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog"
	"gtrade/services/api-gateway/internal/config"
	"gtrade/services/api-gateway/internal/handler"
	httpserver "gtrade/services/api-gateway/internal/http"
	"gtrade/services/api-gateway/internal/repository"
)

func Run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := zerolog.New(os.Stdout).With().Timestamp().Str("service", cfg.ServiceName).Logger()

	if cfg.DatabaseURL != "" {
		pool, err := repository.NewPostgresPool(ctx, cfg.DatabaseURL)
		if err != nil {
			return fmt.Errorf("connect postgres: %w", err)
		}
		defer pool.Close()
	} else {
		logger.Warn().Msg("DATABASE_URL is empty, postgres connection skipped")
	}

	h := handler.New(cfg.ServiceName)
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
