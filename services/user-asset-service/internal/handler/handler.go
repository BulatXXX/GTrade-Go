package handler

import "gtrade/services/user-asset-service/internal/service"

type Handler struct {
	serviceName      string
	userAssetService *service.UserAssetService
}

func New(serviceName string, userAssetService *service.UserAssetService) *Handler {
	return &Handler{serviceName: serviceName, userAssetService: userAssetService}
}
