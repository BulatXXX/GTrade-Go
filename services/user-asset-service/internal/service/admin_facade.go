package service

import (
	"context"
	"time"

	"gtrade/services/user-asset-service/internal/client/catalog"
	"gtrade/services/user-asset-service/internal/model"
	"gtrade/services/user-asset-service/internal/repository"
)

type HandlerFacade struct {
	userAsset  *UserAssetService
	priceAlert *PriceAlertService
}

func NewHandlerFacade(userAsset *UserAssetService, priceAlert *PriceAlertService) *HandlerFacade {
	return &HandlerFacade{
		userAsset:  userAsset,
		priceAlert: priceAlert,
	}
}

func (f *HandlerFacade) CreateUser(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*repository.UserProfile, error) {
	return f.userAsset.CreateUser(ctx, userID, displayName, avatarURL, bio)
}

func (f *HandlerFacade) GetUser(ctx context.Context, userID int64) (*repository.UserProfile, error) {
	return f.userAsset.GetUser(ctx, userID)
}

func (f *HandlerFacade) UpdateUser(ctx context.Context, userID int64, displayName, avatarURL, bio string) (*repository.UserProfile, error) {
	return f.userAsset.UpdateUser(ctx, userID, displayName, avatarURL, bio)
}

func (f *HandlerFacade) ListWatchlist(ctx context.Context, userID int64) ([]repository.WatchlistItem, error) {
	return f.userAsset.ListWatchlist(ctx, userID)
}

func (f *HandlerFacade) AddWatchlistItem(ctx context.Context, userID int64, itemID string) (*repository.WatchlistItem, error) {
	return f.userAsset.AddWatchlistItem(ctx, userID, itemID)
}

func (f *HandlerFacade) UpdateWatchlistNotification(ctx context.Context, userID, watchlistID int64, notifyEnabled bool) (*repository.WatchlistItem, error) {
	return f.userAsset.UpdateWatchlistNotification(ctx, userID, watchlistID, notifyEnabled)
}

func (f *HandlerFacade) DeleteWatchlistItem(ctx context.Context, userID, watchlistID int64) (bool, error) {
	return f.userAsset.DeleteWatchlistItem(ctx, userID, watchlistID)
}

func (f *HandlerFacade) ListRecent(ctx context.Context, userID int64) ([]repository.WatchlistItem, error) {
	return f.userAsset.ListRecent(ctx, userID)
}

func (f *HandlerFacade) GetPreferences(ctx context.Context, userID int64) (*repository.UserPreferences, error) {
	return f.userAsset.GetPreferences(ctx, userID)
}

func (f *HandlerFacade) UpdatePreferences(ctx context.Context, userID int64, currency string, notificationsEnabled bool, notificationMode, notificationTime string) (*repository.UserPreferences, error) {
	return f.userAsset.UpdatePreferences(ctx, userID, currency, notificationsEnabled, notificationMode, notificationTime)
}

func (f *HandlerFacade) GetCatalogItem(ctx context.Context, itemID string) (*catalog.Item, error) {
	return f.userAsset.GetCatalogItem(ctx, itemID)
}

func (f *HandlerFacade) SendManualPriceAlerts(ctx context.Context, userID int64) (*model.AdminManualPriceAlertResponse, error) {
	if f.priceAlert == nil {
		return &model.AdminManualPriceAlertResponse{TargetUserID: userID}, nil
	}
	result, err := f.priceAlert.SendManualPriceAlerts(ctx, userID, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return &model.AdminManualPriceAlertResponse{
		TargetUserID:  result.TargetUserID,
		UsersChecked:  result.UsersChecked,
		EmailsSent:    result.EmailsSent,
		ChangesFound:  result.ChangesFound,
		UsersWithDiff: result.UsersWithDiff,
	}, nil
}

func (f *HandlerFacade) SendAdminMessage(ctx context.Context, userID int64, subject, htmlBody, textBody string) (*model.AdminSendMessageResponse, error) {
	if f.priceAlert == nil {
		return &model.AdminSendMessageResponse{TargetUserID: userID}, nil
	}
	result, err := f.priceAlert.SendAdminMessage(ctx, userID, subject, htmlBody, textBody)
	if err != nil {
		return nil, err
	}
	return &model.AdminSendMessageResponse{
		TargetUserID: result.TargetUserID,
		UsersChecked: result.UsersChecked,
		EmailsSent:   result.EmailsSent,
	}, nil
}
