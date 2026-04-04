package service

import (
	"context"
	"errors"
	"strings"

	"gtrade/services/catalog-service/internal/model"
)

var (
	ErrNotFound       = errors.New("item not found")
	ErrConflict       = errors.New("item already exists")
	ErrInvalidInput   = errors.New("invalid item input")
	ErrNotImplemented = errors.New("catalog service not implemented")
)

type Repository interface {
	CreateItem(ctx context.Context, input model.CreateItemInput) (*model.Item, error)
	UpsertItem(ctx context.Context, input model.CreateItemInput) (*model.Item, error)
	UpdateItem(ctx context.Context, id string, input model.UpdateItemInput) (*model.Item, error)
	DeactivateItem(ctx context.Context, id string) error
	GetItemByID(ctx context.Context, id string) (*model.Item, error)
	ListItems(ctx context.Context, filter model.ListItemsFilter) ([]model.Item, error)
	SearchItems(ctx context.Context, filter model.SearchItemsFilter) ([]model.Item, error)
}

type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateItem(ctx context.Context, input model.CreateItemInput) (*model.Item, error) {
	if err := validateCreateInput(input); err != nil {
		return nil, err
	}

	item, err := s.repo.CreateItem(ctx, normalizeCreateInput(input))
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Service) UpsertItem(ctx context.Context, input model.CreateItemInput) (*model.Item, error) {
	if err := validateCreateInput(input); err != nil {
		return nil, err
	}

	item, err := s.repo.UpsertItem(ctx, normalizeCreateInput(input))
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Service) UpdateItem(ctx context.Context, id string, input model.UpdateItemInput) (*model.Item, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrInvalidInput
	}
	if err := validateUpdateInput(input); err != nil {
		return nil, err
	}

	item, err := s.repo.UpdateItem(ctx, strings.TrimSpace(id), normalizeUpdateInput(input))
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Service) DeleteItem(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return ErrInvalidInput
	}
	return s.repo.DeactivateItem(ctx, strings.TrimSpace(id))
}

func (s *Service) GetItemByID(ctx context.Context, id string) (*model.Item, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrInvalidInput
	}
	return s.repo.GetItemByID(ctx, strings.TrimSpace(id))
}

func (s *Service) ListItems(ctx context.Context, filter model.ListItemsFilter) ([]model.Item, error) {
	if filter.Limit < 0 || filter.Offset < 0 {
		return nil, ErrInvalidInput
	}
	if filter.ActiveOnly == nil {
		activeOnly := true
		filter.ActiveOnly = &activeOnly
	}
	return s.repo.ListItems(ctx, filter)
}

func (s *Service) SearchItems(ctx context.Context, filter model.SearchItemsFilter) ([]model.Item, error) {
	if strings.TrimSpace(filter.Query) == "" || filter.Limit < 0 || filter.Offset < 0 {
		return nil, ErrInvalidInput
	}
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Game = strings.TrimSpace(filter.Game)
	filter.Language = strings.TrimSpace(filter.Language)
	if filter.ActiveOnly == nil {
		activeOnly := true
		filter.ActiveOnly = &activeOnly
	}
	return s.repo.SearchItems(ctx, filter)
}

func validateCreateInput(input model.CreateItemInput) error {
	if strings.TrimSpace(input.Game) == "" ||
		strings.TrimSpace(input.Source) == "" ||
		strings.TrimSpace(input.ExternalID) == "" ||
		strings.TrimSpace(input.Slug) == "" ||
		strings.TrimSpace(input.Name) == "" {
		return ErrInvalidInput
	}
	for _, translation := range input.Translations {
		if err := validateTranslation(translation); err != nil {
			return err
		}
	}
	return nil
}

func validateUpdateInput(input model.UpdateItemInput) error {
	for _, translation := range input.Translations {
		if err := validateTranslation(translation); err != nil {
			return err
		}
	}
	return nil
}

func validateTranslation(translation model.ItemTranslation) error {
	if strings.TrimSpace(translation.LanguageCode) == "" || strings.TrimSpace(translation.Name) == "" {
		return ErrInvalidInput
	}
	return nil
}

func normalizeCreateInput(input model.CreateItemInput) model.CreateItemInput {
	input.Game = strings.TrimSpace(input.Game)
	input.Source = strings.TrimSpace(input.Source)
	input.ExternalID = strings.TrimSpace(input.ExternalID)
	input.Slug = strings.TrimSpace(input.Slug)
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.ImageURL = strings.TrimSpace(input.ImageURL)
	input.Translations = normalizeTranslations(input.Translations)
	return input
}

func normalizeUpdateInput(input model.UpdateItemInput) model.UpdateItemInput {
	input.Slug = strings.TrimSpace(input.Slug)
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.ImageURL = strings.TrimSpace(input.ImageURL)
	input.Translations = normalizeTranslations(input.Translations)
	return input
}

func normalizeTranslations(translations []model.ItemTranslation) []model.ItemTranslation {
	if len(translations) == 0 {
		return nil
	}
	normalized := make([]model.ItemTranslation, 0, len(translations))
	for _, translation := range translations {
		normalized = append(normalized, model.ItemTranslation{
			ItemID:       strings.TrimSpace(translation.ItemID),
			LanguageCode: strings.TrimSpace(translation.LanguageCode),
			Name:         strings.TrimSpace(translation.Name),
			Description:  strings.TrimSpace(translation.Description),
		})
	}
	return normalized
}
