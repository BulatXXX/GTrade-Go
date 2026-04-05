package model

import "time"

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type SearchItemsQuery struct {
	Game     string
	GameMode string
	Query    string
	Limit    int
	Offset   int
}

type GetItemQuery struct {
	Game     string
	GameMode string
	ID       string
}

type GetPricingQuery struct {
	Game     string
	GameMode string
	ID       string
}

type SearchItemsResponse struct {
	Items  []Item `json:"items"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type ItemResponse struct {
	Item Item `json:"item"`
}

type PriceResponse struct {
	Price PriceSnapshot `json:"price"`
}

type Item struct {
	ID          string   `json:"id"`
	Game        string   `json:"game"`
	GameMode    string   `json:"game_mode,omitempty"`
	Source      string   `json:"source"`
	Name        string   `json:"name"`
	Slug        string   `json:"slug,omitempty"`
	Description string   `json:"description,omitempty"`
	ImageURL    string   `json:"image_url,omitempty"`
	URL         string   `json:"url,omitempty"`
	Currency    string   `json:"currency,omitempty"`
	Types       []string `json:"types,omitempty"`
}

type PriceSnapshot struct {
	ItemID     string          `json:"item_id"`
	Game       string          `json:"game"`
	GameMode   string          `json:"game_mode,omitempty"`
	Source     string          `json:"source"`
	Currency   string          `json:"currency"`
	MarketKind string          `json:"market_kind"`
	FetchedAt  time.Time       `json:"fetched_at"`
	Pricing    Pricing         `json:"pricing"`
	Analytics  PricingAnalysis `json:"analytics"`
	RawContext map[string]any  `json:"raw_context,omitempty"`
}

type Pricing struct {
	Current          *float64 `json:"current,omitempty"`
	TopSell          *float64 `json:"top_sell,omitempty"`
	TopBuy           *float64 `json:"top_buy,omitempty"`
	Avg24h           *float64 `json:"avg_24h,omitempty"`
	Low24h           *float64 `json:"low_24h,omitempty"`
	High24h          *float64 `json:"high_24h,omitempty"`
	BasePrice        *float64 `json:"base_price,omitempty"`
	AdjustedPrice    *float64 `json:"adjusted_price,omitempty"`
	Change48hPercent *float64 `json:"change_48h_percent,omitempty"`
	Spread           *float64 `json:"spread,omitempty"`
}

type PricingAnalysis struct {
	SampleSize *int     `json:"sample_size,omitempty"`
	Low        *float64 `json:"low,omitempty"`
	High       *float64 `json:"high,omitempty"`
	Median     *float64 `json:"median,omitempty"`
	Volume24h  *float64 `json:"volume_24h,omitempty"`
}
