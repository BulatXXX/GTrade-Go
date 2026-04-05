package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gtrade/services/api-integration-service/internal/model"
	"gtrade/services/api-integration-service/internal/service/marketplace"
)

var (
	ErrInvalidInput    = errors.New("invalid input")
	ErrUnsupportedGame = errors.New("unsupported game")
	ErrNotFound        = errors.New("not found")
	ErrUpstreamFailed  = errors.New("upstream failed")
)

type Service struct {
	providers map[string]marketplace.Provider
}

func New(providers ...marketplace.Provider) *Service {
	registry := make(map[string]marketplace.Provider, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		registry[strings.ToLower(provider.Game())] = provider
	}

	return &Service{providers: registry}
}

func (s *Service) SearchItems(ctx context.Context, query model.SearchItemsQuery) ([]model.Item, error) {
	if strings.TrimSpace(query.Game) == "" || strings.TrimSpace(query.Query) == "" {
		return nil, ErrInvalidInput
	}
	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Offset < 0 {
		return nil, ErrInvalidInput
	}

	provider, err := s.providerFor(query.Game)
	if err != nil {
		return nil, err
	}

	items, err := provider.SearchItems(ctx, query)
	if err != nil {
		return nil, wrapProviderError(err)
	}

	return items, nil
}

func (s *Service) GetItem(ctx context.Context, query model.GetItemQuery) (*model.Item, error) {
	if strings.TrimSpace(query.Game) == "" || strings.TrimSpace(query.ID) == "" {
		return nil, ErrInvalidInput
	}

	provider, err := s.providerFor(query.Game)
	if err != nil {
		return nil, err
	}

	item, err := provider.GetItem(ctx, query)
	if err != nil {
		return nil, wrapProviderError(err)
	}

	return item, nil
}

func (s *Service) GetPricing(ctx context.Context, query model.GetPricingQuery) (*model.PriceSnapshot, error) {
	if strings.TrimSpace(query.Game) == "" || strings.TrimSpace(query.ID) == "" {
		return nil, ErrInvalidInput
	}

	provider, err := s.providerFor(query.Game)
	if err != nil {
		return nil, err
	}

	price, err := provider.GetPricing(ctx, query)
	if err != nil {
		return nil, wrapProviderError(err)
	}

	return price, nil
}

func (s *Service) providerFor(game string) (marketplace.Provider, error) {
	provider, ok := s.providers[strings.ToLower(strings.TrimSpace(game))]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedGame, game)
	}
	return provider, nil
}

func wrapProviderError(err error) error {
	switch {
	case errors.Is(err, marketplace.ErrNotFound):
		return ErrNotFound
	case errors.Is(err, marketplace.ErrInvalidInput):
		return ErrInvalidInput
	default:
		return fmt.Errorf("%w: %v", ErrUpstreamFailed, err)
	}
}
