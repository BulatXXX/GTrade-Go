package repository

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gtrade/tools/catalog-importer/internal/transform"
)

type Repository interface {
	Upsert(items []transform.Item) error
}

type CatalogHTTPRepository struct {
	client  *http.Client
	baseURL string
	dryRun  bool
}

func NewCatalogHTTPRepository(baseURL string, dryRun bool) *CatalogHTTPRepository {
	return &CatalogHTTPRepository{
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: strings.TrimRight(baseURL, "/"),
		dryRun:  dryRun,
	}
}

func (r *CatalogHTTPRepository) Upsert(items []transform.Item) error {
	for _, item := range items {
		if r.dryRun {
			continue
		}

		payload := map[string]any{
			"game":        item.Game,
			"source":      item.Source,
			"external_id": item.ExternalID,
			"slug":        item.Slug,
			"name":        item.Name,
			"description": item.Description,
			"image_url":   item.ImageURL,
		}

		if len(item.Translations) > 0 {
			payload["translations"] = item.Translations
		}

		body, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal upsert payload: %w", err)
		}

		req, err := http.NewRequest(http.MethodPost, r.baseURL+"/items/upsert", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("build upsert request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := r.client.Do(req)
		if err != nil {
			return fmt.Errorf("request catalog upsert: %w", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("catalog upsert failed: status=%d", resp.StatusCode)
		}
	}
	return nil
}
