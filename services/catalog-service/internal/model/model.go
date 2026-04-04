package model

import "time"

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type Item struct {
	ID                   string            `json:"id"`
	Game                 string            `json:"game"`
	Source               string            `json:"source"`
	ExternalID           string            `json:"external_id"`
	Slug                 string            `json:"slug"`
	Name                 string            `json:"name"`
	LocalizedName        string            `json:"localized_name,omitempty"`
	Description          string            `json:"description,omitempty"`
	LocalizedDescription string            `json:"localized_description,omitempty"`
	LocalizedLanguage    string            `json:"localized_language,omitempty"`
	ImageURL             string            `json:"image_url,omitempty"`
	IsActive             bool              `json:"is_active"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
	Translations         []ItemTranslation `json:"translations,omitempty"`
}

type ItemTranslation struct {
	ItemID       string `json:"item_id,omitempty"`
	LanguageCode string `json:"language_code"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
}

type CreateItemInput struct {
	Game         string
	Source       string
	ExternalID   string
	Slug         string
	Name         string
	Description  string
	ImageURL     string
	Translations []ItemTranslation
}

type UpdateItemInput struct {
	Slug         string
	Name         string
	Description  string
	ImageURL     string
	IsActive     *bool
	Translations []ItemTranslation
}

type ListItemsFilter struct {
	Game       string
	Source     string
	ActiveOnly *bool
	Limit      int
	Offset     int
}

type SearchItemsFilter struct {
	Query      string
	Game       string
	Language   string
	ActiveOnly *bool
	Limit      int
	Offset     int
}

type CreateItemRequest struct {
	Game         string            `json:"game"`
	Source       string            `json:"source"`
	ExternalID   string            `json:"external_id"`
	Slug         string            `json:"slug"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	ImageURL     string            `json:"image_url"`
	Translations []ItemTranslation `json:"translations"`
}

type UpdateItemRequest struct {
	Slug         string            `json:"slug"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	ImageURL     string            `json:"image_url"`
	IsActive     *bool             `json:"is_active"`
	Translations []ItemTranslation `json:"translations"`
}

type ItemResponse struct {
	Item Item `json:"item"`
}

type ListItemsResponse struct {
	Items  []Item `json:"items"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type DeleteItemResponse struct {
	Status string `json:"status"`
}
