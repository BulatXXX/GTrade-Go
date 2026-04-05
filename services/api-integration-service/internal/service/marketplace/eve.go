package marketplace

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gtrade/services/api-integration-service/internal/model"
)

const (
	eveAPIBaseURL   = "https://esi.evetech.net/latest"
	eveImageBaseURL = "https://images.evetech.net/types"
)

type EVEClient struct {
	baseURL      string
	imageBaseURL string
	httpClient   *http.Client
}

func NewEVEClient() *EVEClient {
	return NewEVEClientWithBaseURL(eveAPIBaseURL, eveImageBaseURL, &http.Client{
		Timeout: 10 * time.Second,
	})
}

func NewEVEClientWithBaseURL(baseURL, imageBaseURL string, httpClient *http.Client) *EVEClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	return &EVEClient{
		baseURL:      strings.TrimRight(baseURL, "/"),
		imageBaseURL: strings.TrimRight(imageBaseURL, "/"),
		httpClient:   httpClient,
	}
}

func (c *EVEClient) Game() string { return "eve" }

func (c *EVEClient) SearchItems(_ context.Context, _ model.SearchItemsQuery) ([]model.Item, error) {
	// EVE item search should use the local catalog. ESI pricing endpoints are runtime-only.
	return []model.Item{}, nil
}

func (c *EVEClient) GetItem(ctx context.Context, query model.GetItemQuery) (*model.Item, error) {
	if strings.TrimSpace(query.ID) == "" {
		return nil, ErrInvalidInput
	}

	var resp eveTypeResponse
	if err := c.getJSON(ctx, "/universe/types/"+query.ID+"/?datasource=tranquility", &resp); err != nil {
		return nil, err
	}

	return &model.Item{
		ID:          query.ID,
		Game:        "eve",
		Source:      "esi",
		Name:        resp.Name,
		Description: resp.Description,
		ImageURL:    fmt.Sprintf("%s/%s/icon?size=128", c.imageBaseURL, query.ID),
		URL:         "",
		Currency:    "ISK",
	}, nil
}

func (c *EVEClient) GetPricing(ctx context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error) {
	if strings.TrimSpace(query.ID) == "" {
		return nil, ErrInvalidInput
	}

	var prices []eveMarketPrice
	if err := c.getJSON(ctx, "/markets/prices/?datasource=tranquility", &prices); err != nil {
		return nil, err
	}

	record, ok := findEVEPrice(query.ID, prices)
	if !ok {
		return nil, ErrNotFound
	}

	return &model.PriceSnapshot{
		ItemID:     query.ID,
		Game:       "eve",
		Source:     "esi",
		Currency:   "ISK",
		MarketKind: "reference_market",
		FetchedAt:  time.Now().UTC(),
		Pricing: model.Pricing{
			Current:       record.AveragePrice,
			AdjustedPrice: record.AdjustedPrice,
		},
	}, nil
}

func (c *EVEClient) getJSON(ctx context.Context, requestPath string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+requestPath, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return ErrNotFound
	default:
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

func findEVEPrice(typeID string, prices []eveMarketPrice) (eveMarketPrice, bool) {
	wantID, err := strconv.ParseInt(typeID, 10, 64)
	if err != nil {
		return eveMarketPrice{}, false
	}
	for _, price := range prices {
		if price.TypeID == wantID {
			return price, true
		}
	}
	return eveMarketPrice{}, false
}

type eveTypeResponse struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type eveMarketPrice struct {
	TypeID        int64    `json:"type_id"`
	AdjustedPrice *float64 `json:"adjusted_price"`
	AveragePrice  *float64 `json:"average_price"`
}
