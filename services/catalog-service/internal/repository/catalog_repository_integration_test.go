package repository_test

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"gtrade/services/catalog-service/internal/model"
	"gtrade/services/catalog-service/internal/repository"
)

func TestCatalogRepositoryIntegration_CreateGetUpdateListDeactivate(t *testing.T) {
	ctx := context.Background()
	pool := newCatalogTestPool(t, ctx)
	repo := repository.NewCatalogRepository(pool)

	created, err := repo.CreateItem(ctx, model.CreateItemInput{
		Game:        "warframe",
		Source:      "market",
		ExternalID:  "primed-continuity",
		Slug:        "primed-continuity",
		Name:        "Primed Continuity",
		Description: "Mod that increases ability duration.",
		ImageURL:    "https://cdn.example.com/items/primed-continuity.png",
		Translations: []model.ItemTranslation{
			{
				LanguageCode: "ru",
				Name:         "Праймед Континуити",
				Description:  "Модификация, увеличивающая длительность способностей.",
			},
		},
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}
	if created == nil || created.ID == "" {
		t.Fatalf("created item = %#v", created)
	}

	got, err := repo.GetItemByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get item by id: %v", err)
	}
	if got == nil {
		t.Fatal("expected item to be returned")
	}
	if got.Game != "warframe" || got.Source != "market" || got.ExternalID != "primed-continuity" {
		t.Fatalf("got item = %#v", got)
	}
	if len(got.Translations) != 1 || got.Translations[0].LanguageCode != "ru" {
		t.Fatalf("got translations = %#v", got.Translations)
	}

	updated, err := repo.UpdateItem(ctx, created.ID, model.UpdateItemInput{
		Name:        "Primed Continuity Updated",
		Description: "Updated description",
		ImageURL:    "https://cdn.example.com/items/primed-continuity-updated.png",
		Translations: []model.ItemTranslation{
			{
				LanguageCode: "ru",
				Name:         "Праймед Континуити Обновленный",
				Description:  "Обновленное описание.",
			},
		},
	})
	if err != nil {
		t.Fatalf("update item: %v", err)
	}
	if updated == nil || updated.Name != "Primed Continuity Updated" {
		t.Fatalf("updated item = %#v", updated)
	}

	items, err := repo.ListItems(ctx, model.ListItemsFilter{
		Game:   "warframe",
		Source: "market",
		Limit:  20,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items len = %d, want 1", len(items))
	}

	if err := repo.DeactivateItem(ctx, created.ID); err != nil {
		t.Fatalf("deactivate item: %v", err)
	}

	got, err = repo.GetItemByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get deactivated item: %v", err)
	}
	if got == nil || got.IsActive {
		t.Fatalf("expected item to be deactivated, got %#v", got)
	}
}

func TestCatalogRepositoryIntegration_RejectsDuplicateGameSourceExternalID(t *testing.T) {
	ctx := context.Background()
	pool := newCatalogTestPool(t, ctx)
	repo := repository.NewCatalogRepository(pool)

	input := model.CreateItemInput{
		Game:       "warframe",
		Source:     "market",
		ExternalID: "primed-flow",
		Slug:       "primed-flow",
		Name:       "Primed Flow",
	}

	if _, err := repo.CreateItem(ctx, input); err != nil {
		t.Fatalf("initial create: %v", err)
	}

	_, err := repo.CreateItem(ctx, input)
	if err == nil {
		t.Fatal("expected duplicate create to fail")
	}
}

func TestCatalogRepositoryIntegration_SearchItems_ByBaseNameAndTranslation(t *testing.T) {
	ctx := context.Background()
	pool := newCatalogTestPool(t, ctx)
	repo := repository.NewCatalogRepository(pool)

	activeItem, err := repo.CreateItem(ctx, model.CreateItemInput{
		Game:       "test",
		Source:     "market",
		ExternalID: "primed-continuity-search",
		Slug:       "primed-continuity-search",
		Name:       "Primed Continuity Search",
		Translations: []model.ItemTranslation{
			{
				LanguageCode: "ru",
				Name:         "Праймед Континуити Поиск",
				Description:  "Поиск по переводу",
			},
		},
	})
	if err != nil {
		t.Fatalf("create active item: %v", err)
	}

	inactiveItem, err := repo.CreateItem(ctx, model.CreateItemInput{
		Game:       "test",
		Source:     "market",
		ExternalID: "continuity-inactive-search",
		Slug:       "continuity-inactive-search",
		Name:       "Continuity Inactive Search",
	})
	if err != nil {
		t.Fatalf("create inactive item: %v", err)
	}
	if err := repo.DeactivateItem(ctx, inactiveItem.ID); err != nil {
		t.Fatalf("deactivate inactive item: %v", err)
	}

	items, err := repo.SearchItems(ctx, model.SearchItemsFilter{
		Query:  "continuity",
		Game:   "test",
		Limit:  20,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("search by base name: %v", err)
	}
	if len(items) != 1 || items[0].ID != activeItem.ID {
		t.Fatalf("search by base name returned %#v", items)
	}

	items, err = repo.SearchItems(ctx, model.SearchItemsFilter{
		Query:    "континуити",
		Game:     "test",
		Language: "ru",
		Limit:    20,
		Offset:   0,
	})
	if err != nil {
		t.Fatalf("search by translation: %v", err)
	}
	if len(items) != 1 || items[0].ID != activeItem.ID {
		t.Fatalf("search by translation returned %#v", items)
	}
}

func TestCatalogRepositoryIntegration_UpsertItem_UpdatesExistingRecord(t *testing.T) {
	ctx := context.Background()
	pool := newCatalogTestPool(t, ctx)
	repo := repository.NewCatalogRepository(pool)

	first, err := repo.UpsertItem(ctx, model.CreateItemInput{
		Game:       "test",
		Source:     "market",
		ExternalID: "upsert-item",
		Slug:       "upsert-item",
		Name:       "Upsert Item",
		Translations: []model.ItemTranslation{
			{LanguageCode: "ru", Name: "Апсерт Предмет"},
		},
	})
	if err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	second, err := repo.UpsertItem(ctx, model.CreateItemInput{
		Game:       "test",
		Source:     "market",
		ExternalID: "upsert-item",
		Slug:       "upsert-item-updated",
		Name:       "Upsert Item Updated",
		Translations: []model.ItemTranslation{
			{LanguageCode: "ru", Name: "Апсерт Предмет Обновлен"},
		},
	})
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	if first.ID != second.ID {
		t.Fatalf("upsert should update same record, ids: %q vs %q", first.ID, second.ID)
	}
	if second.Name != "Upsert Item Updated" || second.Slug != "upsert-item-updated" {
		t.Fatalf("updated item = %#v", second)
	}
	if len(second.Translations) != 1 || second.Translations[0].Name != "Апсерт Предмет Обновлен" {
		t.Fatalf("updated translations = %#v", second.Translations)
	}
}

func TestCatalogRepositoryIntegration_UpsertItem_PreservesTranslationsWhenOmitted(t *testing.T) {
	ctx := context.Background()
	pool := newCatalogTestPool(t, ctx)
	repo := repository.NewCatalogRepository(pool)

	first, err := repo.UpsertItem(ctx, model.CreateItemInput{
		Game:       "test",
		Source:     "market",
		ExternalID: "preserve-translations",
		Slug:       "preserve-translations",
		Name:       "Preserve Item",
		Translations: []model.ItemTranslation{
			{LanguageCode: "ru", Name: "Сохранить Перевод", Description: "Описание"},
		},
	})
	if err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	second, err := repo.UpsertItem(ctx, model.CreateItemInput{
		Game:       "test",
		Source:     "market",
		ExternalID: "preserve-translations",
		Slug:       "preserve-translations-updated",
		Name:       "Preserve Item Updated",
	})
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	if first.ID != second.ID {
		t.Fatalf("upsert should update same record, ids: %q vs %q", first.ID, second.ID)
	}
	if len(second.Translations) != 1 || second.Translations[0].LanguageCode != "ru" || second.Translations[0].Name != "Сохранить Перевод" {
		t.Fatalf("translations should be preserved, got %#v", second.Translations)
	}
}

func TestCatalogRepositoryIntegration_UpsertAndReadPriceHistory(t *testing.T) {
	ctx := context.Background()
	pool := newCatalogTestPool(t, ctx)
	repo := repository.NewCatalogRepository(pool)

	item, err := repo.CreateItem(ctx, model.CreateItemInput{
		Game:       "tarkov",
		Source:     "tarkov-dev",
		ExternalID: "5448bd6b4bdc2dfc2f8b4569",
		Slug:       "5448bd6b4bdc2dfc2f8b4569",
		Name:       "Makarov PM",
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	firstCollectedAt := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	if err := repo.UpsertPriceHistory(ctx, model.UpsertPriceHistoryInput{
		ItemID:      item.ID,
		Source:      "tarkov-dev",
		GameMode:    "regular",
		Value:       15000,
		Currency:    "RUB",
		CollectedOn: firstCollectedAt,
		CollectedAt: firstCollectedAt,
	}); err != nil {
		t.Fatalf("initial upsert price history: %v", err)
	}

	secondCollectedAt := time.Date(2026, 5, 2, 18, 30, 0, 0, time.UTC)
	if err := repo.UpsertPriceHistory(ctx, model.UpsertPriceHistoryInput{
		ItemID:      item.ID,
		Source:      "tarkov-dev",
		GameMode:    "regular",
		Value:       16000,
		Currency:    "RUB",
		CollectedOn: secondCollectedAt,
		CollectedAt: secondCollectedAt,
	}); err != nil {
		t.Fatalf("second upsert price history: %v", err)
	}

	otherDayCollectedAt := time.Date(2026, 5, 3, 9, 0, 0, 0, time.UTC)
	if err := repo.UpsertPriceHistory(ctx, model.UpsertPriceHistoryInput{
		ItemID:      item.ID,
		Source:      "tarkov-dev",
		GameMode:    "pve",
		Value:       22000,
		Currency:    "RUB",
		CollectedOn: otherDayCollectedAt,
		CollectedAt: otherDayCollectedAt,
	}); err != nil {
		t.Fatalf("third upsert price history: %v", err)
	}

	regularHistory, err := repo.GetPriceHistory(ctx, item.ID, model.PriceHistoryFilter{
		GameMode: "regular",
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("get regular price history: %v", err)
	}
	if len(regularHistory) != 1 {
		t.Fatalf("regular history len = %d, want 1", len(regularHistory))
	}
	if regularHistory[0].Value != 16000 || regularHistory[0].CollectedOn != "2026-05-02" {
		t.Fatalf("regular history row = %#v", regularHistory[0])
	}

	allHistory, err := repo.GetPriceHistory(ctx, item.ID, model.PriceHistoryFilter{Limit: 10})
	if err != nil {
		t.Fatalf("get all price history: %v", err)
	}
	if len(allHistory) != 2 {
		t.Fatalf("all history len = %d, want 2", len(allHistory))
	}
	if allHistory[0].GameMode != "pve" || allHistory[1].GameMode != "regular" {
		t.Fatalf("unexpected history ordering/content: %#v", allHistory)
	}
}

func newCatalogTestPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	pool, err := repository.NewPostgresPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect test postgres: %v", err)
	}

	t.Cleanup(pool.Close)

	applyCatalogMigrations(t, ctx, pool.Exec)

	if _, err := pool.Exec(ctx, `TRUNCATE TABLE item_translations, prices, items RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate catalog tables: %v", err)
	}

	return pool
}

func applyCatalogMigrations(
	t *testing.T,
	ctx context.Context,
	execFn func(context.Context, string, ...any) (pgconn.CommandTag, error),
) {
	t.Helper()

	migrationPaths, err := filepath.Glob(filepath.Join("..", "..", "migrations", "*.sql"))
	if err != nil {
		t.Fatalf("glob migration files: %v", err)
	}
	sort.Strings(migrationPaths)

	for _, migrationPath := range migrationPaths {
		migrationSQL, err := os.ReadFile(migrationPath)
		if err != nil {
			t.Fatalf("read migration file %s: %v", migrationPath, err)
		}

		statements := strings.Split(string(migrationSQL), ";")
		for _, statement := range statements {
			statement = strings.TrimSpace(statement)
			if statement == "" {
				continue
			}
			if _, err := execFn(ctx, statement); err != nil {
				t.Fatalf("apply migration %s statement %q: %v", migrationPath, statement, err)
			}
		}
	}
}
