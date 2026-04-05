package service

import (
	"context"
	"fmt"
	"strings"

	"gtrade/services/user-asset-service/internal/repository"
)

type UserAssetService struct {
	repo userAssetRepository
}

type userAssetRepository interface {
	CreateUser(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*repository.UserProfile, error)
	GetUser(ctx context.Context, userID int64) (*repository.UserProfile, error)
	UpdateUser(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*repository.UserProfile, error)
	ListWatchlist(ctx context.Context, userID int64) ([]repository.WatchlistItem, error)
	AddWatchlistItem(ctx context.Context, userID int64, itemID string) (*repository.WatchlistItem, error)
	DeleteWatchlistItem(ctx context.Context, userID, watchlistID int64) (bool, error)
	ListRecent(ctx context.Context, userID int64, limit int) ([]repository.WatchlistItem, error)
	GetPreferences(ctx context.Context, userID int64) (*repository.UserPreferences, error)
	UpsertPreferences(ctx context.Context, userID int64, currency string, notificationsEnabled bool) (*repository.UserPreferences, error)
}

func NewUserAssetService(repo userAssetRepository) *UserAssetService {
	return &UserAssetService{repo: repo}
}

func (s *UserAssetService) CreateUser(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*repository.UserProfile, error) {
	if userID <= 0 || displayName == "" {
		return nil, fmt.Errorf("user_id and display_name are required")
	}
	return s.repo.CreateUser(ctx, userID, strings.TrimSpace(displayName), strings.TrimSpace(avatarURL), strings.TrimSpace(bio))
}

func (s *UserAssetService) GetUser(ctx context.Context, userID int64) (*repository.UserProfile, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("user_id is required")
	}
	return s.repo.GetUser(ctx, userID)
}

func (s *UserAssetService) UpdateUser(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*repository.UserProfile, error) {
	if userID <= 0 || strings.TrimSpace(displayName) == "" {
		return nil, fmt.Errorf("user_id and display_name are required")
	}
	return s.repo.UpdateUser(ctx, userID, strings.TrimSpace(displayName), strings.TrimSpace(avatarURL), strings.TrimSpace(bio))
}

func (s *UserAssetService) ListWatchlist(ctx context.Context, userID int64) ([]repository.WatchlistItem, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("user_id is required")
	}
	return s.repo.ListWatchlist(ctx, userID)
}

func (s *UserAssetService) AddWatchlistItem(ctx context.Context, userID int64, itemID string) (*repository.WatchlistItem, error) {
	if userID <= 0 || strings.TrimSpace(itemID) == "" {
		return nil, fmt.Errorf("user_id and item_id are required")
	}
	return s.repo.AddWatchlistItem(ctx, userID, strings.TrimSpace(itemID))
}

func (s *UserAssetService) DeleteWatchlistItem(ctx context.Context, userID, watchlistID int64) (bool, error) {
	if userID <= 0 || watchlistID <= 0 {
		return false, fmt.Errorf("user_id and watchlist id are required")
	}
	return s.repo.DeleteWatchlistItem(ctx, userID, watchlistID)
}

func (s *UserAssetService) ListRecent(ctx context.Context, userID int64) ([]repository.WatchlistItem, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("user_id is required")
	}
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
