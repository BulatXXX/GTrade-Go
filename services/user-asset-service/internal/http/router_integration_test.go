package http

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"gtrade/services/user-asset-service/internal/client/catalog"
	"gtrade/services/user-asset-service/internal/handler"
	"gtrade/services/user-asset-service/internal/repository"
	"gtrade/services/user-asset-service/internal/service"
	"net/http"
	"net/http/httptest"
)

type fakeCatalogClient struct {
	items map[string]*catalog.Item
}

func (f fakeCatalogClient) GetItem(ctx context.Context, id string) (*catalog.Item, error) {
	item, ok := f.items[id]
	if !ok {
		return nil, catalog.ErrNotFound
	}
	return item, nil
}

func TestRouterIntegration_WatchlistIsValidatedAndEnrichedByCatalog(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is required")
	}

	ctx := context.Background()
	pool, err := repository.NewPostgresPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	applyUserAssetMigrations(t, ctx, pool)
	resetUserAssetTables(t, ctx, pool)

	repo := repository.NewUserAssetRepository(pool)
	catalogClient := fakeCatalogClient{
		items: map[string]*catalog.Item{
			"item-uuid-1": {
				ID:       "item-uuid-1",
				Game:     "warframe",
				Source:   "market",
				Name:     "Frost Prime Set",
				Slug:     "frost_prime_set",
				ImageURL: "https://cdn.example.com/frost.png",
				IsActive: true,
			},
		},
	}
	svc := service.NewUserAssetService(repo, catalogClient)
	router := NewRouter(zerolog.Nop(), handler.New("user-asset-service", svc))

	createUserReq := newIntegrationJSONRequest(t, http.MethodPost, "/users", map[string]any{
		"user_id":      101,
		"display_name": "Alice",
		"avatar_url":   "https://cdn.example.com/a.png",
		"bio":          "Trader",
	})
	createUserRec := httptest.NewRecorder()
	router.ServeHTTP(createUserRec, createUserReq)
	if createUserRec.Code != http.StatusCreated {
		t.Fatalf("create user status = %d; body=%s", createUserRec.Code, createUserRec.Body.String())
	}

	addWatchlistReq := newIntegrationJSONRequest(t, http.MethodPost, "/watchlist", map[string]any{
		"user_id": 101,
		"item_id": "item-uuid-1",
	})
	addWatchlistRec := httptest.NewRecorder()
	router.ServeHTTP(addWatchlistRec, addWatchlistReq)
	if addWatchlistRec.Code != http.StatusCreated {
		t.Fatalf("add watchlist status = %d; body=%s", addWatchlistRec.Code, addWatchlistRec.Body.String())
	}

	var createdItem map[string]any
	if err := json.Unmarshal(addWatchlistRec.Body.Bytes(), &createdItem); err != nil {
		t.Fatalf("unmarshal created watchlist item: %v", err)
	}
	itemData, ok := createdItem["item"].(map[string]any)
	if !ok || itemData["name"] != "Frost Prime Set" {
		t.Fatalf("expected enriched item in create response, got %v", createdItem)
	}

	getWatchlistReq := httptest.NewRequest(http.MethodGet, "/watchlist?user_id=101", nil)
	getWatchlistRec := httptest.NewRecorder()
	router.ServeHTTP(getWatchlistRec, getWatchlistReq)
	if getWatchlistRec.Code != http.StatusOK {
		t.Fatalf("get watchlist status = %d; body=%s", getWatchlistRec.Code, getWatchlistRec.Body.String())
	}

	var watchlistBody map[string]any
	if err := json.Unmarshal(getWatchlistRec.Body.Bytes(), &watchlistBody); err != nil {
		t.Fatalf("unmarshal watchlist: %v", err)
	}
	items, ok := watchlistBody["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("unexpected watchlist body: %v", watchlistBody)
	}
	firstItem := items[0].(map[string]any)
	enriched := firstItem["item"].(map[string]any)
	if enriched["slug"] != "frost_prime_set" {
		t.Fatalf("unexpected enriched item: %v", enriched)
	}
}

func TestRouterIntegration_RejectsMissingCatalogItem(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is required")
	}

	ctx := context.Background()
	pool, err := repository.NewPostgresPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	applyUserAssetMigrations(t, ctx, pool)
	resetUserAssetTables(t, ctx, pool)

	repo := repository.NewUserAssetRepository(pool)
	svc := service.NewUserAssetService(repo, fakeCatalogClient{items: map[string]*catalog.Item{}})
	router := NewRouter(zerolog.Nop(), handler.New("user-asset-service", svc))

	createUserReq := newIntegrationJSONRequest(t, http.MethodPost, "/users", map[string]any{
		"user_id":      202,
		"display_name": "Alice",
	})
	createUserRec := httptest.NewRecorder()
	router.ServeHTTP(createUserRec, createUserReq)
	if createUserRec.Code != http.StatusCreated {
		t.Fatalf("create user status = %d; body=%s", createUserRec.Code, createUserRec.Body.String())
	}

	addWatchlistReq := newIntegrationJSONRequest(t, http.MethodPost, "/watchlist", map[string]any{
		"user_id": 202,
		"item_id": "missing-item",
	})
	addWatchlistRec := httptest.NewRecorder()
	router.ServeHTTP(addWatchlistRec, addWatchlistReq)
	if addWatchlistRec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d; body=%s", addWatchlistRec.Code, addWatchlistRec.Body.String())
	}
}

func newIntegrationJSONRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func applyUserAssetMigrations(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	root := filepath.Join("..", "..")
	migrations := []string{
		filepath.Join(root, "migrations", "0001_init.sql"),
		filepath.Join(root, "migrations", "0002_profile_fields_and_watchlist_refs.sql"),
	}

	for _, migration := range migrations {
		sql, err := os.ReadFile(migration)
		if err != nil {
			t.Fatalf("read migration %s: %v", migration, err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("apply migration %s: %v", migration, err)
		}
	}
}

func resetUserAssetTables(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
		TRUNCATE TABLE watchlist_items, user_preferences, user_profiles RESTART IDENTITY;
	`); err != nil {
		t.Fatalf("reset tables: %v", err)
	}
}
