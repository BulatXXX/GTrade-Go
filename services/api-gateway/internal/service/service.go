package service

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrInvalidTarget = errors.New("invalid target")
	ErrUpstream      = errors.New("upstream request failed")
)

const (
	TargetAuth         = "auth"
	TargetUserAsset    = "user-asset"
	TargetCatalog      = "catalog"
	TargetIntegration  = "integration"
	TargetNotification = "notification"
)

type Targets struct {
	AuthURL         string
	UserAssetURL    string
	CatalogURL      string
	IntegrationURL  string
	NotificationURL string
}

type Service struct {
	client  ServiceClient
	targets map[string]string
}

func New(client ServiceClient, targets Targets) *Service {
	return &Service{
		client: client,
		targets: map[string]string{
			TargetAuth:         targets.AuthURL,
			TargetUserAsset:    targets.UserAssetURL,
			TargetCatalog:      targets.CatalogURL,
			TargetIntegration:  targets.IntegrationURL,
			TargetNotification: targets.NotificationURL,
		},
	}
}

func (s *Service) Forward(ctx context.Context, target string, req ForwardRequest) (*ForwardResponse, error) {
	baseURL, ok := s.targets[target]
	if !ok || baseURL == "" {
		return nil, fmt.Errorf("%w: %s", ErrInvalidTarget, target)
	}

	resp, err := s.client.Forward(ctx, baseURL, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}

	return resp, nil
}
