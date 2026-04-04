package service

import (
	"context"
	"fmt"

	"gtrade/services/user-asset-service/internal/repository"
)

type UserAssetService struct {
	repo *repository.UserAssetRepository
}

func NewUserAssetService(repo *repository.UserAssetRepository) *UserAssetService {
	return &UserAssetService{repo: repo}
}

func (s *UserAssetService) CreateUser(ctx context.Context, userID int64, displayName string) (*repository.UserProfile, error) {
	if userID <= 0 || displayName == "" {
		return nil, fmt.Errorf("user_id and display_name are required")
	}
	return s.repo.CreateUser(ctx, userID, displayName)
}

func (s *UserAssetService) GetUser(ctx context.Context, userID int64) (*repository.UserProfile, error) {
	return s.repo.GetUser(ctx, userID)
}

func (s *UserAssetService) ListWatchlist(ctx context.Context, userID int64) ([]repository.WatchlistItem, error) {
	return s.repo.ListWatchlist(ctx, userID)
}

func (s *UserAssetService) AddWatchlistItem(ctx context.Context, userID, itemID int64) (*repository.WatchlistItem, error) {
	if userID <= 0 || itemID <= 0 {
		return nil, fmt.Errorf("user_id and item_id are required")
	}
	return s.repo.AddWatchlistItem(ctx, userID, itemID)
}

func (s *UserAssetService) DeleteWatchlistItem(ctx context.Context, userID, watchlistID int64) (bool, error) {
	if userID <= 0 || watchlistID <= 0 {
		return false, fmt.Errorf("user_id and watchlist id are required")
	}
	return s.repo.DeleteWatchlistItem(ctx, userID, watchlistID)
}

func (s *UserAssetService) ListRecent(ctx context.Context, userID int64) ([]repository.WatchlistItem, error) {
	return s.repo.ListRecent(ctx, userID, 10)
}

func (s *UserAssetService) GetPreferences(ctx context.Context, userID int64) (*repository.UserPreferences, error) {
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}
	if prefs == nil {
		return s.repo.UpsertPreferences(ctx, userID, "credits", true)
	}
	return prefs, nil
}

func (s *UserAssetService) UpdatePreferences(ctx context.Context, userID int64, currency string, notificationsEnabled bool) (*repository.UserPreferences, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("user_id is required")
	}
	if currency == "" {
		currency = "credits"
	}
	return s.repo.UpsertPreferences(ctx, userID, currency, notificationsEnabled)
}
