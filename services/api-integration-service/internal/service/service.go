package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	catalogclient "gtrade/services/api-integration-service/internal/client/catalog"
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
	providers     map[string]marketplace.Provider
	catalogWriter CatalogWriter
}

func New(providers ...marketplace.Provider) *Service {
	return NewWithCatalog(nil, providers...)
}

type CatalogWriter interface {
	UpsertItem(ctx context.Context, input catalogclient.UpsertItemRequest) (*catalogclient.Item, error)
}

func NewWithCatalog(catalogWriter CatalogWriter, providers ...marketplace.Provider) *Service {
	registry := make(map[string]marketplace.Provider, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		registry[strings.ToLower(provider.Game())] = provider
	}

	return &Service{
		providers:     registry,
		catalogWriter: catalogWriter,
	}
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
	query.GameMode = normalizeGameMode(query.Game, query.GameMode)

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
	query.GameMode = normalizeGameMode(query.Game, query.GameMode)

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
	query.GameMode = normalizeGameMode(query.Game, query.GameMode)

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

func (s *Service) SyncItemToCatalog(ctx context.Context, query model.SyncItemQuery) (*model.SyncedCatalogItem, error) {
	if s.catalogWriter == nil {
		return nil, ErrUpstreamFailed
	}
	if strings.TrimSpace(query.Game) == "" || strings.TrimSpace(query.ID) == "" {
		return nil, ErrInvalidInput
	}

	item, err := s.GetItem(ctx, model.GetItemQuery{
		Game:     query.Game,
		GameMode: query.GameMode,
		ID:       query.ID,
	})
	if err != nil {
		return nil, err
	}

	synced, err := s.catalogWriter.UpsertItem(ctx, toCatalogUpsert(*item))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstreamFailed, err)
	}

	return toSyncedCatalogItem(*synced), nil
}

func (s *Service) SyncSearchToCatalog(ctx context.Context, query model.SyncSearchQuery) ([]model.SyncedCatalogItem, error) {
	if s.catalogWriter == nil {
		return nil, ErrUpstreamFailed
	}
	if strings.TrimSpace(query.Game) == "" || strings.TrimSpace(query.Query) == "" {
		return nil, ErrInvalidInput
	}

	items, err := s.SearchItems(ctx, model.SearchItemsQuery{
		Game:     query.Game,
		GameMode: query.GameMode,
		Query:    query.Query,
		Limit:    query.Limit,
		Offset:   query.Offset,
	})
	if err != nil {
		return nil, err
	}

	syncedItems := make([]model.SyncedCatalogItem, 0, len(items))
	for _, item := range items {
		synced, err := s.catalogWriter.UpsertItem(ctx, toCatalogUpsert(item))
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrUpstreamFailed, err)
		}
		syncedItems = append(syncedItems, *toSyncedCatalogItem(*synced))
	}

	return syncedItems, nil
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

func normalizeGameMode(game, gameMode string) string {
	if strings.ToLower(strings.TrimSpace(game)) != "tarkov" {
		return strings.TrimSpace(gameMode)
	}

	mode := strings.ToLower(strings.TrimSpace(gameMode))
	if mode == "" {
		return "regular"
	}
	return mode
}

func toCatalogUpsert(item model.Item) catalogclient.UpsertItemRequest {
	slug := strings.TrimSpace(item.Slug)
	if slug == "" {
		slug = strings.TrimSpace(item.ID)
	}

	return catalogclient.UpsertItemRequest{
		Game:        strings.TrimSpace(item.Game),
		Source:      strings.TrimSpace(item.Source),
		ExternalID:  strings.TrimSpace(item.ID),
		Slug:        slug,
		Name:        strings.TrimSpace(item.Name),
		Description: strings.TrimSpace(item.Description),
		ImageURL:    strings.TrimSpace(item.ImageURL),
	}
}

func toSyncedCatalogItem(item catalogclient.Item) *model.SyncedCatalogItem {
	return &model.SyncedCatalogItem{
		ID:         item.ID,
		Game:       item.Game,
		Source:     item.Source,
		ExternalID: item.ExternalID,
		Slug:       item.Slug,
		Name:       item.Name,
		ImageURL:   item.ImageURL,
		UpdatedAt:  item.UpdatedAt,
	}
}
