package marketplace

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gtrade/services/api-integration-service/internal/model"
)

const tarkovAPIURL = "https://api.tarkov.dev/graphql"

type TarkovClient struct {
	endpoint   string
	httpClient *http.Client
}

func NewTarkovClient() *TarkovClient {
	return NewTarkovClientWithBaseURL(tarkovAPIURL, &http.Client{
		Timeout: 10 * time.Second,
	})
}

func NewTarkovClientWithBaseURL(endpoint string, httpClient *http.Client) *TarkovClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	return &TarkovClient{
		endpoint:   strings.TrimRight(endpoint, "/"),
		httpClient: httpClient,
	}
}

func (c *TarkovClient) Game() string { return "tarkov" }

func (c *TarkovClient) SearchItems(ctx context.Context, query model.SearchItemsQuery) ([]model.Item, error) {
	if strings.TrimSpace(query.Query) == "" {
		return nil, ErrInvalidInput
	}

	var resp tarkovSearchResponse
	err := c.postGraphQL(ctx, tarkovGraphQLRequest{
		Query: `
query SearchItems($name: String!, $gameMode: GameMode!, $offset: Int!, $limit: Int!) {
  items(name: $name, gameMode: $gameMode, offset: $offset, limit: $limit) {
    id
    name
    shortName
    types
    description
    avg24hPrice
    low24hPrice
    high24hPrice
    basePrice
    changeLast48hPercent
    width
    height
    iconLink
    image512pxLink
    link
    sellFor {
      price
      source
      currency
    }
  }
}`,
		Variables: map[string]any{
			"name":     query.Query,
			"gameMode": query.GameMode,
			"offset":   query.Offset,
			"limit":    query.Limit,
		},
	}, &resp)
	if err != nil {
		return nil, err
	}

	items := make([]model.Item, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, toTarkovItem(item, query.GameMode))
	}
	return items, nil
}

func (c *TarkovClient) GetItem(ctx context.Context, query model.GetItemQuery) (*model.Item, error) {
	if strings.TrimSpace(query.ID) == "" {
		return nil, ErrInvalidInput
	}

	item, err := c.fetchItem(ctx, query.ID, query.GameMode)
	if err != nil {
		return nil, err
	}

	mapped := toTarkovItem(item, query.GameMode)
	return &mapped, nil
}

func (c *TarkovClient) GetPricing(ctx context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error) {
	if strings.TrimSpace(query.ID) == "" {
		return nil, ErrInvalidInput
	}

	item, err := c.fetchItem(ctx, query.ID, query.GameMode)
	if err != nil {
		return nil, err
	}

	bestTraderPrice, bestTraderSource := bestTraderSellFor(item.SellFor)
	fleaPrice := fleaMarketSellFor(item.SellFor)

	return &model.PriceSnapshot{
		ItemID:     query.ID,
		Game:       "tarkov",
		GameMode:   query.GameMode,
		Source:     "tarkov-dev",
		Currency:   "RUB",
		MarketKind: "aggregated_market",
		FetchedAt:  time.Now().UTC(),
		Pricing: model.Pricing{
			Current:          intPtrToFloat64(item.Avg24hPrice),
			Avg24h:           intPtrToFloat64(item.Avg24hPrice),
			Low24h:           intPtrToFloat64(item.Low24hPrice),
			High24h:          intPtrToFloat64(item.High24hPrice),
			BasePrice:        intPtrToFloat64(item.BasePrice),
			Change48hPercent: item.ChangeLast48hPercent,
			TopSell:          fleaPrice,
		},
		Analytics: model.PricingAnalysis{
			Low:  intPtrToFloat64(item.Low24hPrice),
			High: intPtrToFloat64(item.High24hPrice),
		},
		RawContext: map[string]any{
			"best_trader_price":  bestTraderPrice,
			"best_trader_source": bestTraderSource,
			"flea_market_price":  fleaPrice,
		},
	}, nil
}

func (c *TarkovClient) fetchItem(ctx context.Context, id, gameMode string) (tarkovItem, error) {
	var resp tarkovItemResponse
	err := c.postGraphQL(ctx, tarkovGraphQLRequest{
		Query: `
query ItemByID($id: ID!, $gameMode: GameMode!) {
  item(id: $id, gameMode: $gameMode) {
    id
    name
    shortName
    types
    description
    avg24hPrice
    low24hPrice
    high24hPrice
    basePrice
    changeLast48hPercent
    width
    height
    iconLink
    image512pxLink
    link
    sellFor {
      price
      source
      currency
    }
  }
}`,
		Variables: map[string]any{
			"id":       id,
			"gameMode": gameMode,
		},
	}, &resp)
	if err != nil {
		return tarkovItem{}, err
	}
	if resp.Item.ID == "" {
		return tarkovItem{}, ErrNotFound
	}
	return resp.Item, nil
}

func (c *TarkovClient) postGraphQL(ctx context.Context, payload tarkovGraphQLRequest, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal graphql request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var envelope struct {
		Data   json.RawMessage      `json:"data"`
		Errors []tarkovGraphQLError `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if len(envelope.Errors) > 0 {
		return fmt.Errorf("graphql request failed: %w", errors.Join(toTarkovGraphQLErrors(envelope.Errors)...))
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return fmt.Errorf("graphql response missing data")
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return fmt.Errorf("decode graphql data: %w", err)
	}
	return nil
}

func toTarkovItem(item tarkovItem, gameMode string) model.Item {
	imageURL := item.Image512pxLink
	if strings.TrimSpace(imageURL) == "" {
		imageURL = item.IconLink
	}

	return model.Item{
		ID:          item.ID,
		Game:        "tarkov",
		GameMode:    gameMode,
		Source:      "tarkov-dev",
		Name:        item.Name,
		Slug:        item.ID,
		Description: item.Description,
		ImageURL:    imageURL,
		URL:         item.Link,
		Currency:    "RUB",
		Types:       item.Types,
	}
}

func intPtrToFloat64(v *int) *float64 {
	if v == nil {
		return nil
	}
	value := float64(*v)
	return &value
}

func bestTraderSellFor(values []tarkovSellFor) (*float64, string) {
	var best *float64
	bestSource := ""
	for _, value := range values {
		if strings.EqualFold(value.Source, "fleaMarket") {
			continue
		}
		price := float64(value.Price)
		if best == nil || price > *best {
			best = &price
			bestSource = value.Source
		}
	}
	return best, bestSource
}

func fleaMarketSellFor(values []tarkovSellFor) *float64 {
	for _, value := range values {
		if strings.EqualFold(value.Source, "fleaMarket") {
			price := float64(value.Price)
			return &price
		}
	}
	return nil
}

type tarkovGraphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type tarkovGraphQLError struct {
	Message string `json:"message"`
}

type tarkovSearchResponse struct {
	Items []tarkovItem `json:"items"`
}

type tarkovItemResponse struct {
	Item tarkovItem `json:"item"`
}

type tarkovItem struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name"`
	ShortName            string          `json:"shortName"`
	Types                []string        `json:"types"`
	Description          string          `json:"description"`
	Avg24hPrice          *int            `json:"avg24hPrice"`
	Low24hPrice          *int            `json:"low24hPrice"`
	High24hPrice         *int            `json:"high24hPrice"`
	BasePrice            *int            `json:"basePrice"`
	ChangeLast48hPercent *float64        `json:"changeLast48hPercent"`
	Width                *int            `json:"width"`
	Height               *int            `json:"height"`
	IconLink             string          `json:"iconLink"`
	Image512pxLink       string          `json:"image512pxLink"`
	Link                 string          `json:"link"`
	SellFor              []tarkovSellFor `json:"sellFor"`
}

type tarkovSellFor struct {
	Price    int    `json:"price"`
	Source   string `json:"source"`
	Currency string `json:"currency"`
}

func (e tarkovGraphQLError) Error() string {
	return strings.TrimSpace(e.Message)
}

func toTarkovGraphQLErrors(values []tarkovGraphQLError) []error {
	errs := make([]error, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value.Message) == "" {
			continue
		}
		errs = append(errs, value)
	}
	if len(errs) == 0 {
		errs = append(errs, errors.New("unknown graphql error"))
	}
	return errs
}
