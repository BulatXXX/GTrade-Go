package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
