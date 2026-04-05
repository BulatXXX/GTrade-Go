package marketplace

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"gtrade/services/api-integration-service/internal/model"
)

const (
	warframeAPIBaseURL    = "https://api.warframe.market/v2"
	warframeAssetsBaseURL = "https://warframe.market/static/assets"
	warframeItemPageURL   = "https://warframe.market/items"
)

type WarframeClient struct {
	baseURL         string
	assetsBaseURL   string
	itemPageBaseURL string
	httpClient      *http.Client
}

func NewWarframeClient() *WarframeClient {
	return NewWarframeClientWithBaseURL(warframeAPIBaseURL, warframeAssetsBaseURL, warframeItemPageURL, &http.Client{
		Timeout: 10 * time.Second,
	})
}

func NewWarframeClientWithBaseURL(baseURL, assetsBaseURL, itemPageBaseURL string, httpClient *http.Client) *WarframeClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	return &WarframeClient{
		baseURL:         strings.TrimRight(baseURL, "/"),
		assetsBaseURL:   strings.TrimRight(assetsBaseURL, "/"),
		itemPageBaseURL: strings.TrimRight(itemPageBaseURL, "/"),
		httpClient:      httpClient,
	}
}

func (c *WarframeClient) Game() string { return "warframe" }

func (c *WarframeClient) SearchItems(ctx context.Context, query model.SearchItemsQuery) ([]model.Item, error) {
	if strings.TrimSpace(query.Query) == "" {
		return nil, ErrInvalidInput
	}

	var resp warframeItemsListResponse
	if err := c.getJSON(ctx, "/items", &resp); err != nil {
		return nil, err
	}

	searchTerm := strings.ToLower(strings.TrimSpace(query.Query))
	start := clampOffset(query.Offset)
	limit := clampLimit(query.Limit)

	filtered := make([]model.Item, 0, limit)
	skipped := 0
	for _, item := range resp.Data {
		name := strings.TrimSpace(item.I18n.EN.Name)
		slug := strings.TrimSpace(item.Slug)
		if !matchesWarframeQuery(name, slug, searchTerm) {
			continue
		}
		if skipped < start {
			skipped++
			continue
		}

		filtered = append(filtered, c.toItem(item))
		if len(filtered) >= limit {
			break
		}
	}

	return filtered, nil
}

func (c *WarframeClient) GetItem(ctx context.Context, query model.GetItemQuery) (*model.Item, error) {
	if strings.TrimSpace(query.ID) == "" {
		return nil, ErrInvalidInput
	}

	var resp warframeItemResponse
	if err := c.getJSON(ctx, "/items/"+url.PathEscape(query.ID), &resp); err != nil {
		return nil, err
	}

	item := c.toItem(resp.Data)
	return &item, nil
}

func (c *WarframeClient) GetPricing(ctx context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error) {
	if strings.TrimSpace(query.ID) == "" {
		return nil, ErrInvalidInput
	}

	var resp warframeTopOrdersResponse
	if err := c.getJSON(ctx, "/orders/item/"+url.PathEscape(query.ID)+"/top", &resp); err != nil {
		return nil, err
	}

	topSell := firstWarframeOrderPrice(resp.Data.Sell)
	topBuy := firstWarframeOrderPrice(resp.Data.Buy)
	spread := computeSpread(topSell, topBuy)
	low := minWarframeOrderPrice(resp.Data.Sell)
	high := maxWarframeOrderPrice(resp.Data.Sell)
	sampleSize := len(resp.Data.Sell) + len(resp.Data.Buy)

	return &model.PriceSnapshot{
		ItemID:     query.ID,
		Game:       "warframe",
		Source:     "warframe-market",
		Currency:   "PLAT",
		MarketKind: "live_orders",
		FetchedAt:  time.Now().UTC(),
		Pricing: model.Pricing{
			Current: topSell,
			TopSell: topSell,
			TopBuy:  topBuy,
			Spread:  spread,
		},
		Analytics: model.PricingAnalysis{
			SampleSize: intPtr(sampleSize),
			Low:        low,
			High:       high,
		},
		RawContext: map[string]any{
			"sell_order_count": len(resp.Data.Sell),
			"buy_order_count":  len(resp.Data.Buy),
		},
	}, nil
}

func (c *WarframeClient) getJSON(ctx context.Context, requestPath string, out any) error {
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

func (c *WarframeClient) toItem(item warframeItem) model.Item {
	return model.Item{
		ID:          item.Slug,
		Game:        "warframe",
		Source:      "warframe-market",
		Name:        item.I18n.EN.Name,
		Slug:        item.Slug,
		Description: item.I18n.EN.Description,
		ImageURL:    c.absoluteAssetURL(item.I18n.EN.Icon),
		URL:         c.absoluteItemURL(item.Slug),
		Currency:    "PLAT",
		Types:       item.Tags,
	}
}

func (c *WarframeClient) absoluteAssetURL(iconPath string) string {
	if strings.TrimSpace(iconPath) == "" {
		return ""
	}
	if strings.HasPrefix(iconPath, "http://") || strings.HasPrefix(iconPath, "https://") {
		return iconPath
	}
	return c.assetsBaseURL + "/" + strings.TrimLeft(iconPath, "/")
}

func (c *WarframeClient) absoluteItemURL(slug string) string {
	if strings.TrimSpace(slug) == "" {
		return ""
	}
	base, err := url.Parse(c.itemPageBaseURL + "/")
	if err != nil {
		return c.itemPageBaseURL + "/" + slug
	}
	base.Path = path.Join(base.Path, slug)
	return base.String()
}

func matchesWarframeQuery(name, slug, searchTerm string) bool {
	name = strings.ToLower(name)
	slug = strings.ToLower(slug)
	return strings.Contains(name, searchTerm) || strings.Contains(slug, searchTerm)
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func clampOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

func firstWarframeOrderPrice(orders []warframeOrder) *float64 {
	if len(orders) == 0 {
		return nil
	}
	value := float64(orders[0].Platinum)
	return &value
}

func minWarframeOrderPrice(orders []warframeOrder) *float64 {
	if len(orders) == 0 {
		return nil
	}
	min := float64(orders[0].Platinum)
	for _, order := range orders[1:] {
		value := float64(order.Platinum)
		if value < min {
			min = value
		}
	}
	return &min
}

func maxWarframeOrderPrice(orders []warframeOrder) *float64 {
	if len(orders) == 0 {
		return nil
	}
	max := float64(orders[0].Platinum)
	for _, order := range orders[1:] {
		value := float64(order.Platinum)
		if value > max {
			max = value
		}
	}
	return &max
}

func computeSpread(topSell, topBuy *float64) *float64 {
	if topSell == nil || topBuy == nil {
		return nil
	}
	spread := *topSell - *topBuy
	return &spread
}

func intPtr(v int) *int { return &v }

type warframeItemsListResponse struct {
	Data []warframeItem `json:"data"`
}

type warframeItemResponse struct {
	Data warframeItem `json:"data"`
}

type warframeTopOrdersResponse struct {
	Data warframeTopOrdersData `json:"data"`
}

type warframeTopOrdersData struct {
	Sell []warframeOrder `json:"sell"`
	Buy  []warframeOrder `json:"buy"`
}

type warframeOrder struct {
	Platinum int `json:"platinum"`
}

type warframeItem struct {
	Slug string           `json:"slug"`
	Tags []string         `json:"tags"`
	I18n warframeItemI18n `json:"i18n"`
}

type warframeItemI18n struct {
	EN warframeItemLocale `json:"en"`
}

type warframeItemLocale struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}
