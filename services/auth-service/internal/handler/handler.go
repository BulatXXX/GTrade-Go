package handler

import (
	"context"

	"gtrade/services/auth-service/internal/service"
)

type AuthUseCase interface {
	Register(ctx context.Context, email, password string) (*service.TokenPair, error)
	Login(ctx context.Context, email, password string) (*service.TokenPair, error)
	Refresh(ctx context.Context, refreshToken string) (*service.TokenPair, error)
}

type Handler struct {
	serviceName string
	authService AuthUseCase
}

func New(serviceName string, authService AuthUseCase) *Handler {
	return &Handler{serviceName: serviceName, authService: authService}
}
