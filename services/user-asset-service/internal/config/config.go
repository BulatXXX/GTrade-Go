package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	ServiceName  string
	Port         int
	DatabaseURL  string
	JWTSecret    string
	LogLevel     string
	ResendAPIKey string
}

func Load() (Config, error) {
	port, err := getEnvInt("PORT", 8080)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		ServiceName:  getEnv("SERVICE_NAME", "service"),
		Port:         port,
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		JWTSecret:    getEnv("JWT_SECRET", "change-me"),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		ResendAPIKey: getEnv("RESEND_API_KEY", ""),
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
