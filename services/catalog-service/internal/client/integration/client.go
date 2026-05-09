package integration

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

var ErrNotFound = errors.New("integration price not found")

type TopPrice struct {
	ItemID    string    `json:"item_id"`
	Game      string    `json:"game"`
	GameMode  string    `json:"game_mode,omitempty"`
	Source    string    `json:"source"`
	Currency  string    `json:"currency"`
	Value     *float64  `json:"value,omitempty"`
	FetchedAt time.Time `json:"fetched_at"`
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

func (c *Client) GetTopPrice(ctx context.Context, externalID, game, gameMode string) (*TopPrice, error) {
	endpoint, err := url.Parse(c.baseURL + "/items/" + url.PathEscape(strings.TrimSpace(externalID)) + "/top-price")
	if err != nil {
		return nil, fmt.Errorf("build integration url: %w", err)
	}

	query := endpoint.Query()
	query.Set("game", strings.TrimSpace(game))
	if strings.TrimSpace(gameMode) != "" {
		query.Set("game_mode", strings.TrimSpace(gameMode))
	}
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build integration request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("integration request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, ErrNotFound
	default:
		return nil, fmt.Errorf("integration unexpected status: %d", resp.StatusCode)
	}

	var out TopPrice
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode integration top price: %w", err)
	}

	return &out, nil
}
