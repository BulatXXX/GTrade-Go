package model

type CreateUserRequest struct {
	UserID      int64  `json:"user_id"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Bio         string `json:"bio"`
}

type UpdateUserRequest struct {
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Bio         string `json:"bio"`
}

type AddWatchlistRequest struct {
	UserID int64  `json:"user_id"`
	ItemID string `json:"item_id"`
}

type UpdatePreferencesRequest struct {
	UserID               int64  `json:"user_id"`
	Currency             string `json:"currency"`
	NotificationsEnabled bool   `json:"notifications_enabled"`
}

type UserProfileResponse struct {
	UserID      int64  `json:"user_id"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Bio         string `json:"bio"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type CatalogItemSummary struct {
	ID       string `json:"id"`
	Game     string `json:"game"`
	Source   string `json:"source"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	ImageURL string `json:"image_url"`
}

type WatchlistItemResponse struct {
	ID        int64               `json:"id"`
	UserID    int64               `json:"user_id"`
	ItemID    string              `json:"item_id"`
	Item      *CatalogItemSummary `json:"item,omitempty"`
	CreatedAt string              `json:"created_at"`
}

type PreferencesResponse struct {
	UserID               int64  `json:"user_id"`
	Currency             string `json:"currency"`
	NotificationsEnabled bool   `json:"notifications_enabled"`
	UpdatedAt            string `json:"updated_at"`
}
