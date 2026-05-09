package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrNotFound = errors.New("catalog item not found")

type Item struct {
	ID       string `json:"id"`
	Game     string `json:"game"`
	Source   string `json:"source"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	ImageURL string `json:"image_url"`
	IsActive bool   `json:"is_active"`
}

type PriceHistoryEntry struct {
	ItemID      string    `json:"item_id"`
	Source      string    `json:"source"`
	GameMode    string    `json:"game_mode,omitempty"`
	Value       float64   `json:"value"`
	Currency    string    `json:"currency"`
	CollectedOn string    `json:"collected_on"`
	CollectedAt time.Time `json:"collected_at"`
}

type PriceHistoryResponse struct {
	ItemID   string              `json:"item_id"`
	GameMode string              `json:"game_mode,omitempty"`
	History  []PriceHistoryEntry `json:"history"`
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
			Timeout: 5 * time.Second,
		},
	}
}

func (c *Client) GetItem(ctx context.Context, id string) (*Item, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/items/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("build catalog request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("catalog request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, ErrNotFound
	default:
		return nil, fmt.Errorf("catalog unexpected status: %d", resp.StatusCode)
	}

	var out itemResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode catalog item: %w", err)
	}
	return &out.Item, nil
}

func (c *Client) GetPriceHistory(ctx context.Context, id, gameMode string, limit int) (*PriceHistoryResponse, error) {
	endpoint, err := url.Parse(c.baseURL + "/items/" + url.PathEscape(strings.TrimSpace(id)) + "/prices/history")
	if err != nil {
		return nil, fmt.Errorf("build catalog history url: %w", err)
	}

	query := endpoint.Query()
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	if strings.TrimSpace(gameMode) != "" {
		query.Set("game_mode", strings.TrimSpace(gameMode))
	}
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build catalog history request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("catalog history request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, ErrNotFound
	default:
		return nil, fmt.Errorf("catalog history unexpected status: %d", resp.StatusCode)
	}

	var out PriceHistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode catalog price history: %w", err)
	}
	return &out, nil
}
