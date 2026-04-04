package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"gtrade/services/catalog-service/internal/model"
)

func TestServiceCreateItem_ValidatesRequiredFields(t *testing.T) {
	t.Parallel()

	svc := New(stubRepository{})

	_, err := svc.CreateItem(context.Background(), model.CreateItemInput{})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("create item error = %v, want %v", err, ErrInvalidInput)
	}
}

func TestServiceCreateItem_DelegatesToRepository(t *testing.T) {
	t.Parallel()

	var got model.CreateItemInput
	now := time.Now().UTC()
	repo := stubRepository{
		createItemFn: func(ctx context.Context, input model.CreateItemInput) (*model.Item, error) {
			got = input
			return &model.Item{
				ID:         "item-1",
				Game:       input.Game,
				Source:     input.Source,
				ExternalID: input.ExternalID,
				Slug:       input.Slug,
				Name:       input.Name,
				IsActive:   true,
				CreatedAt:  now,
				UpdatedAt:  now,
			}, nil
		},
	}

	svc := New(repo)
	item, err := svc.CreateItem(context.Background(), model.CreateItemInput{
		Game:       "warframe",
		Source:     "market",
		ExternalID: "primed-continuity",
		Slug:       "primed-continuity",
		Name:       "Primed Continuity",
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	if item == nil || item.ID == "" {
		t.Fatalf("expected created item with id, got %#v", item)
	}
	if got.Game != "warframe" || got.Source != "market" || got.ExternalID != "primed-continuity" {
		t.Fatalf("repository input = %#v", got)
	}
}

func TestServiceUpdateItem_DelegatesToRepository(t *testing.T) {
	t.Parallel()

	var gotID string
	var gotInput model.UpdateItemInput
	repo := stubRepository{
		updateItemFn: func(ctx context.Context, id string, input model.UpdateItemInput) (*model.Item, error) {
			gotID = id
			gotInput = input
			return &model.Item{ID: id, Name: input.Name, Slug: input.Slug, IsActive: true}, nil
		},
	}

	svc := New(repo)
	item, err := svc.UpdateItem(context.Background(), "item-1", model.UpdateItemInput{
		Name: "Primed Continuity Updated",
		Slug: "primed-continuity",
	})
	if err != nil {
		t.Fatalf("update item: %v", err)
	}

	if item == nil || item.ID != "item-1" {
		t.Fatalf("updated item = %#v", item)
	}
	if gotID != "item-1" {
		t.Fatalf("repository id = %q, want %q", gotID, "item-1")
	}
	if gotInput.Name != "Primed Continuity Updated" {
		t.Fatalf("repository input = %#v", gotInput)
	}
}

func TestServiceDeleteItem_DelegatesToRepository(t *testing.T) {
	t.Parallel()

	var gotID string
	repo := stubRepository{
		deactivateItemFn: func(ctx context.Context, id string) error {
			gotID = id
			return nil
		},
	}

	svc := New(repo)
	if err := svc.DeleteItem(context.Background(), "item-1"); err != nil {
		t.Fatalf("delete item: %v", err)
	}
	if gotID != "item-1" {
		t.Fatalf("repository id = %q, want %q", gotID, "item-1")
	}
}

func TestServiceGetItemByID_DelegatesToRepository(t *testing.T) {
	t.Parallel()

	repo := stubRepository{
		getItemByIDFn: func(ctx context.Context, id string) (*model.Item, error) {
			return &model.Item{ID: id, Name: "Primed Continuity", IsActive: true}, nil
		},
	}

	svc := New(repo)
	item, err := svc.GetItemByID(context.Background(), "item-1")
	if err != nil {
		t.Fatalf("get item by id: %v", err)
	}
	if item == nil || item.ID != "item-1" {
		t.Fatalf("item = %#v", item)
	}
}

func TestServiceListItems_DefaultsToActiveOnlyAndDelegates(t *testing.T) {
	t.Parallel()

	var got model.ListItemsFilter
	repo := stubRepository{
		listItemsFn: func(ctx context.Context, filter model.ListItemsFilter) ([]model.Item, error) {
			got = filter
			return []model.Item{{ID: "item-1", Name: "Primed Continuity", IsActive: true}}, nil
		},
	}

	svc := New(repo)
	items, err := svc.ListItems(context.Background(), model.ListItemsFilter{Limit: 20})
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items len = %d, want 1", len(items))
	}
	if got.ActiveOnly == nil || !*got.ActiveOnly {
		t.Fatalf("expected active_only default to true, got %#v", got.ActiveOnly)
	}
}

func TestServiceSearchItems_ValidatesQuery(t *testing.T) {
	t.Parallel()

	svc := New(stubRepository{})

	_, err := svc.SearchItems(context.Background(), model.SearchItemsFilter{})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("search items error = %v, want %v", err, ErrInvalidInput)
	}
}

func TestServiceSearchItems_DefaultsToActiveOnlyAndDelegates(t *testing.T) {
	t.Parallel()

	var got model.SearchItemsFilter
	repo := stubRepository{
		searchItemsFn: func(ctx context.Context, filter model.SearchItemsFilter) ([]model.Item, error) {
			got = filter
			return []model.Item{{ID: "item-1", Name: "Primed Continuity", IsActive: true}}, nil
		},
	}

	svc := New(repo)
	items, err := svc.SearchItems(context.Background(), model.SearchItemsFilter{
		Query:    "continuity",
		Game:     "warframe",
		Language: "ru",
		Limit:    20,
	})
	if err != nil {
		t.Fatalf("search items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items len = %d, want 1", len(items))
	}
	if got.ActiveOnly == nil || !*got.ActiveOnly {
		t.Fatalf("expected active_only default to true, got %#v", got.ActiveOnly)
	}
	if got.Query != "continuity" || got.Game != "warframe" || got.Language != "ru" {
		t.Fatalf("search filter = %#v", got)
	}
}

type stubRepository struct {
	createItemFn     func(ctx context.Context, input model.CreateItemInput) (*model.Item, error)
	updateItemFn     func(ctx context.Context, id string, input model.UpdateItemInput) (*model.Item, error)
	deactivateItemFn func(ctx context.Context, id string) error
	getItemByIDFn    func(ctx context.Context, id string) (*model.Item, error)
	listItemsFn      func(ctx context.Context, filter model.ListItemsFilter) ([]model.Item, error)
	searchItemsFn    func(ctx context.Context, filter model.SearchItemsFilter) ([]model.Item, error)
}

func (s stubRepository) CreateItem(ctx context.Context, input model.CreateItemInput) (*model.Item, error) {
	if s.createItemFn == nil {
		return nil, errors.New("unexpected CreateItem call")
	}
	return s.createItemFn(ctx, input)
}

func (s stubRepository) UpdateItem(ctx context.Context, id string, input model.UpdateItemInput) (*model.Item, error) {
	if s.updateItemFn == nil {
		return nil, errors.New("unexpected UpdateItem call")
	}
	return s.updateItemFn(ctx, id, input)
}

func (s stubRepository) DeactivateItem(ctx context.Context, id string) error {
	if s.deactivateItemFn == nil {
		return errors.New("unexpected DeactivateItem call")
	}
	return s.deactivateItemFn(ctx, id)
}

func (s stubRepository) GetItemByID(ctx context.Context, id string) (*model.Item, error) {
	if s.getItemByIDFn == nil {
		return nil, errors.New("unexpected GetItemByID call")
	}
	return s.getItemByIDFn(ctx, id)
}

func (s stubRepository) ListItems(ctx context.Context, filter model.ListItemsFilter) ([]model.Item, error) {
	if s.listItemsFn == nil {
		return nil, errors.New("unexpected ListItems call")
	}
	return s.listItemsFn(ctx, filter)
}

func (s stubRepository) SearchItems(ctx context.Context, filter model.SearchItemsFilter) ([]model.Item, error) {
	if s.searchItemsFn == nil {
		return nil, errors.New("unexpected SearchItems call")
	}
	return s.searchItemsFn(ctx, filter)
}
