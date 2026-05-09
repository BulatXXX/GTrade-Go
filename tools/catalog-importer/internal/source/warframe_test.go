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
			case "/orders/item/frost_prime_set/top":
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`{
						"data": {"sell":[{"platinum":12}],"buy":[]}
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

func TestWarframeSourceStream_SkipsItemsWithoutSellOrders(t *testing.T) {
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
							{"id":"1","slug":"sellable","i18n":{"en":{"name":"Sellable","icon":"a.png"}}},
							{"id":"2","slug":"dead_item","i18n":{"en":{"name":"Dead","icon":"b.png"}}},
							{"id":"3","slug":"missing_item","i18n":{"en":{"name":"Missing","icon":"c.png"}}}
						]
					}`)),
				}, nil
			case "/orders/item/sellable/top":
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"data":{"sell":[{"platinum":10}],"buy":[]}}`)),
				}, nil
			case "/orders/item/dead_item/top":
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"data":{"sell":[],"buy":[]}}`)),
				}, nil
			case "/orders/item/missing_item/top":
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"error":"not_found"}`)),
				}, nil
			case "/items/sellable":
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`{
						"data": {"id":"1","slug":"sellable","i18n":{"en":{"name":"Sellable","description":"","icon":"a.png"}}}
					}`)),
				}, nil
			default:
				t.Fatalf("unexpected path: %s", req.URL.Path)
			}
			return nil, nil
		}),
	}

	src := NewWarframeSource(client, "en", 10)
	src.baseURL = "https://example.test"

	var items []RawItem
	if err := src.Stream(context.Background(), func(item RawItem) error {
		items = append(items, item)
		return nil
	}); err != nil {
		t.Fatalf("stream items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items len = %d, want 1", len(items))
	}
	if items[0].Slug != "sellable" {
		t.Fatalf("item = %#v", items[0])
	}
}
