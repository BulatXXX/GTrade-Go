package model

type ItemDTO struct {
	ID       string `json:"id"`
	Game     string `json:"game"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
}

type PriceDTO struct {
	ItemID   string  `json:"item_id"`
	Source   string  `json:"source"`
	Value    float64 `json:"value"`
	Currency string  `json:"currency"`
}
