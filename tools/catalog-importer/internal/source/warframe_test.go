package source

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestWarframeSourceFetch_ParsesItemsAndLocalizedTranslation(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/items":
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`{
						"data": [
							{
								"id": "1",
								"slug": "frost_prime_set",
								"i18n": {
									"en": {
										"name": "Frost Prime Set",
										"icon": "items/images/en/frost_prime_set.png"
									}
								}
							}
						]
					}`)),
				}, nil
			case "/items/frost_prime_set":
				if req.Header.Get("Language") == "ru" {
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     make(http.Header),
						Body: io.NopCloser(strings.NewReader(`{
							"data": {
								"id": "1",
								"slug": "frost_prime_set",
								"i18n": {
									"ru": {
										"name": "Набор Фроста Прайм",
										"description": "Перевод описания",
										"icon": "items/images/ru/frost_prime_set.png"
									}
								}
							}
						}`)),
					}, nil
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`{
						"data": {
							"id": "1",
							"slug": "frost_prime_set",
							"i18n": {
								"en": {
									"name": "Frost Prime Set",
									"description": "Base description",
									"icon": "items/images/en/frost_prime_set.png"
								}
							}
						}
					}`)),
				}, nil
			default:
				t.Fatalf("unexpected path: %s", req.URL.Path)
			}
			return nil, nil
		}),
	}

	src := NewWarframeSource(client, "ru", 10)
	src.baseURL = "https://example.test"

	var items []RawItem
	err := src.Stream(context.Background(), func(item RawItem) error {
		items = append(items, item)
		return nil
	})
	if err != nil {
		t.Fatalf("stream items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items len = %d, want 1", len(items))
	}

	item := items[0]
	if item.Game != "warframe" || item.Source != "market" {
		t.Fatalf("item = %#v", item)
	}
	if item.ExternalID != "frost_prime_set" || item.Slug != "frost_prime_set" {
		t.Fatalf("item identifiers = %#v", item)
	}
	if item.Name != "Frost Prime Set" {
		t.Fatalf("base name = %q, want %q", item.Name, "Frost Prime Set")
	}
	if item.Description != "Base description" {
		t.Fatalf("base description = %q", item.Description)
	}
	if item.ImageURL != "https://warframe.market/static/assets/items/images/en/frost_prime_set.png" {
		t.Fatalf("image url = %q", item.ImageURL)
	}
	if len(item.Translations) != 1 {
		t.Fatalf("translations = %#v", item.Translations)
	}
	if item.Translations[0].LanguageCode != "ru" || item.Translations[0].Name != "Набор Фроста Прайм" {
		t.Fatalf("translation = %#v", item.Translations[0])
	}
	if item.Translations[0].Description != "Перевод описания" {
		t.Fatalf("translation description = %#v", item.Translations[0])
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
