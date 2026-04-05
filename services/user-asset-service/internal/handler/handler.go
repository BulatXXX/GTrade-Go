package handler

import (
	"context"

	"gtrade/services/user-asset-service/internal/client/catalog"
	"gtrade/services/user-asset-service/internal/repository"
)

type Handler struct {
	serviceName      string
	userAssetService UserAssetUseCase
}

type UserAssetUseCase interface {
	CreateUser(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*repository.UserProfile, error)
	GetUser(ctx context.Context, userID int64) (*repository.UserProfile, error)
	UpdateUser(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*repository.UserProfile, error)
	ListWatchlist(ctx context.Context, userID int64) ([]repository.WatchlistItem, error)
	AddWatchlistItem(ctx context.Context, userID int64, itemID string) (*repository.WatchlistItem, error)
	DeleteWatchlistItem(ctx context.Context, userID, watchlistID int64) (bool, error)
	ListRecent(ctx context.Context, userID int64) ([]repository.WatchlistItem, error)
	GetPreferences(ctx context.Context, userID int64) (*repository.UserPreferences, error)
	UpdatePreferences(ctx context.Context, userID int64, currency string, notificationsEnabled bool) (*repository.UserPreferences, error)
	GetCatalogItem(ctx context.Context, itemID string) (*catalog.Item, error)
}

func New(serviceName string, userAssetService UserAssetUseCase) *Handler {
	return &Handler{serviceName: serviceName, userAssetService: userAssetService}
}
