package model

type CreateUserRequest struct {
	UserID      int64  `json:"user_id"`
	DisplayName string `json:"display_name"`
}

type AddWatchlistRequest struct {
	UserID int64 `json:"user_id"`
	ItemID int64 `json:"item_id"`
}

type UpdatePreferencesRequest struct {
	UserID               int64  `json:"user_id"`
	Currency             string `json:"currency"`
	NotificationsEnabled bool   `json:"notifications_enabled"`
}

type UserProfileResponse struct {
	UserID      int64  `json:"user_id"`
	DisplayName string `json:"display_name"`
}

type WatchlistItemResponse struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	ItemID    int64  `json:"item_id"`
	CreatedAt string `json:"created_at"`
}

type PreferencesResponse struct {
	UserID               int64  `json:"user_id"`
	Currency             string `json:"currency"`
	NotificationsEnabled bool   `json:"notifications_enabled"`
	UpdatedAt            string `json:"updated_at"`
}
