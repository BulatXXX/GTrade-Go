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

type UpdateWatchlistNotificationsRequest struct {
	UserID        int64 `json:"user_id"`
	NotifyEnabled bool  `json:"notify_enabled"`
}

type UpdatePreferencesRequest struct {
	UserID               int64  `json:"user_id"`
	Currency             string `json:"currency"`
	NotificationsEnabled bool   `json:"notifications_enabled"`
	NotificationMode     string `json:"notification_mode"`
	NotificationTime     string `json:"notification_time"`
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
	ID            int64               `json:"id"`
	UserID        int64               `json:"user_id"`
	ItemID        string              `json:"item_id"`
	NotifyEnabled bool                `json:"notify_enabled"`
	Item          *CatalogItemSummary `json:"item,omitempty"`
	CreatedAt     string              `json:"created_at"`
}

type PreferencesResponse struct {
	UserID               int64  `json:"user_id"`
	Currency             string `json:"currency"`
	NotificationsEnabled bool   `json:"notifications_enabled"`
	NotificationMode     string `json:"notification_mode"`
	NotificationTime     string `json:"notification_time"`
	UpdatedAt            string `json:"updated_at"`
}

type AdminManualPriceAlertRequest struct {
	UserID    int64 `json:"user_id,omitempty"`
	ForceSend bool  `json:"force_send,omitempty"`
}

type AdminManualPriceAlertResponse struct {
	TargetUserID  int64  `json:"target_user_id,omitempty"`
	Mode          string `json:"mode"`
	UsersChecked  int    `json:"users_checked"`
	EmailsSent    int    `json:"emails_sent"`
	ChangesFound  int    `json:"changes_found"`
	UsersWithDiff int    `json:"users_with_diff"`
	UsersSkipped  int    `json:"users_skipped"`
}

type SchedulerStateResponse struct {
	Items []SchedulerStateItem `json:"items"`
}

type SchedulerStateItem struct {
	JobName        string  `json:"job_name"`
	Status         string  `json:"status"`
	LastStartedAt  *string `json:"last_started_at,omitempty"`
	LastFinishedAt *string `json:"last_finished_at,omitempty"`
	LastError      *string `json:"last_error,omitempty"`
	LastProcessed  int     `json:"last_processed"`
	LastTotal      int     `json:"last_total"`
	RunsTotal      int64   `json:"runs_total"`
	UpdatedAt      string  `json:"updated_at"`
}

type AdminSendMessageRequest struct {
	UserID   int64  `json:"user_id,omitempty"`
	Subject  string `json:"subject"`
	HTMLBody string `json:"html_body"`
	TextBody string `json:"text_body"`
}

type AdminSendMessageResponse struct {
	TargetUserID int64 `json:"target_user_id,omitempty"`
	UsersChecked int   `json:"users_checked"`
	EmailsSent   int   `json:"emails_sent"`
}
