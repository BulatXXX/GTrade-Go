package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	ServiceName         string
	Port                int
	DatabaseURL         string
	AuthServiceURL      string
	UserAssetServiceURL string
	CatalogServiceURL   string
	IntegrationURL      string
	NotificationURL     string
	JWTSecret           string
	LogLevel            string
	ResendAPIKey        string
}

func Load() (Config, error) {
	port, err := getEnvInt("PORT", 8080)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		ServiceName:         getEnv("SERVICE_NAME", "service"),
		Port:                port,
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		AuthServiceURL:      getEnv("AUTH_SERVICE_URL", "http://localhost:8081"),
		UserAssetServiceURL: getEnv("USER_ASSET_SERVICE_URL", "http://localhost:8082"),
		CatalogServiceURL:   getEnv("CATALOG_SERVICE_URL", "http://localhost:8084"),
		IntegrationURL:      getEnv("API_INTEGRATION_SERVICE_URL", "http://localhost:8083"),
		NotificationURL:     getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8085"),
		JWTSecret:           getEnv("JWT_SECRET", "change-me"),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		ResendAPIKey:        getEnv("RESEND_API_KEY", ""),
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid int env %s: %w", key, err)
	}

	return parsed, nil
}
