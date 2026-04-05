package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var ErrUpsertFailed = errors.New("catalog upsert failed")

type ItemTranslation struct {
	LanguageCode string `json:"language_code"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
}

type UpsertItemRequest struct {
	Game         string            `json:"game"`
	Source       string            `json:"source"`
	ExternalID   string            `json:"external_id"`
	Slug         string            `json:"slug"`
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	ImageURL     string            `json:"image_url,omitempty"`
	Translations []ItemTranslation `json:"translations,omitempty"`
}

type Item struct {
	ID         string    `json:"id"`
	Game       string    `json:"game"`
	Source     string    `json:"source"`
	ExternalID string    `json:"external_id"`
	Slug       string    `json:"slug"`
	Name       string    `json:"name"`
	ImageURL   string    `json:"image_url,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type itemResponse struct {
	Item Item `json:"item"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) UpsertItem(ctx context.Context, input UpsertItemRequest) (*Item, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal catalog upsert: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/items/upsert", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build catalog request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("catalog request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrUpsertFailed, resp.StatusCode)
	}

	var out itemResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode catalog upsert: %w", err)
	}

	return &out.Item, nil
}
