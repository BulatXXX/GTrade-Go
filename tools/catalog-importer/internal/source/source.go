package source

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

var ErrNotImplemented = errors.New("source is not implemented yet")

type RawTranslation struct {
	LanguageCode string
	Name         string
	Description  string
}

type RawItem struct {
	Game         string
	Source       string
	ExternalID   string
	Slug         string
	Name         string
	Description  string
	ImageURL     string
	Translations []RawTranslation
}

type Source interface {
	Stream(ctx context.Context, consume func(RawItem) error) error
}

type Config struct {
	Name       string
	Language   string
	Limit      int
	HTTPClient *http.Client
}

func New(cfg Config) (Source, error) {
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	switch cfg.Name {
	case "warframe":
		return NewWarframeSource(client, cfg.Language, cfg.Limit), nil
	case "eve":
		return NewEVESource(client, cfg.Language, cfg.Limit), nil
	case "tarkov":
		return NewTarkovSource(), nil
	default:
		return nil, fmt.Errorf("unsupported source: %s", cfg.Name)
	}
}
