package repository

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"gtrade/tools/catalog-importer/internal/transform"
)

func TestCatalogHTTPRepositoryUpsert_PostsItemsToCatalogService(t *testing.T) {
	t.Parallel()

	var received []string
	repo := NewCatalogHTTPRepository("https://catalog.test", false)
	repo.client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost || req.URL.Path != "/items/upsert" {
				t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			received = append(received, string(body))

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"item":{"id":"item-1"}}`)),
			}, nil
		}),
	}

	err := repo.Upsert([]transform.Item{{
		Game:       "warframe",
		Source:     "market",
		ExternalID: "frost_prime_set",
		Slug:       "frost_prime_set",
		Name:       "Frost Prime Set",
		Translations: []transform.Translation{
			{LanguageCode: "ru", Name: "Набор Фроста Прайм"},
		},
	}})
	if err != nil {
		t.Fatalf("upsert items: %v", err)
	}

	if len(received) != 1 {
		t.Fatalf("received requests = %d, want 1", len(received))
	}
	if !strings.Contains(received[0], `"game":"warframe"`) || !strings.Contains(received[0], `"source":"market"`) {
		t.Fatalf("payload = %s", received[0])
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
