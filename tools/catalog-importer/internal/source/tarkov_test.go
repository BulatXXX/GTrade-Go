package source

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestTarkovSourceStream_ParsesItemsAndLocalizedTranslation(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			payload := string(body)

			if req.Method != http.MethodPost || req.URL.Path != "/graphql" {
				t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
			}

			switch {
			case strings.Contains(payload, "lang: en") && strings.Contains(payload, "offset: 0") && strings.Contains(payload, "limit: 2"):
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`{
						"data": {
							"items": [
								{
									"id": "item-1",
									"name": "AK test item",
									"description": "Base description",
									"iconLink": "https://assets.tarkov.dev/item-1-icon.webp",
									"image512pxLink": "https://assets.tarkov.dev/item-1-512.webp"
								},
								{
									"id": "item-2",
									"name": "Second item",
									"description": "",
									"iconLink": "https://assets.tarkov.dev/item-2-icon.webp",
									"image512pxLink": ""
								}
							]
						}
					}`)),
				}, nil
			case strings.Contains(payload, "lang: ru") && strings.Contains(payload, "offset: 0") && strings.Contains(payload, "limit: 2"):
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`{
						"data": {
							"items": [
								{
									"id": "item-1",
									"name": "АК тестовый предмет",
									"description": "Русское описание",
									"iconLink": "https://assets.tarkov.dev/item-1-icon.webp",
									"image512pxLink": "https://assets.tarkov.dev/item-1-512.webp"
								},
								{
									"id": "item-2",
									"name": "Второй предмет",
									"description": "",
									"iconLink": "https://assets.tarkov.dev/item-2-icon.webp",
									"image512pxLink": ""
								}
							]
						}
					}`)),
				}, nil
			case strings.Contains(payload, "lang: en") && strings.Contains(payload, "offset: 2"):
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"data":{"items":[]}}`)),
				}, nil
			case strings.Contains(payload, "lang: ru") && strings.Contains(payload, "offset: 2"):
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"data":{"items":[]}}`)),
				}, nil
			default:
				t.Fatalf("unexpected request body: %s", payload)
			}

			return nil, nil
		}),
	}

	src := NewTarkovSource(client, "ru", 2)
	src.baseURL = "https://example.test/graphql"

	var items []RawItem
	err := src.Stream(context.Background(), func(item RawItem) error {
		items = append(items, item)
		return nil
	})
	if err != nil {
		t.Fatalf("stream items: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("items len = %d, want 2", len(items))
	}

	first := items[0]
	if first.Game != "tarkov" || first.Source != "tarkov-dev" {
		t.Fatalf("first item = %#v", first)
	}
	if first.ExternalID != "item-1" || first.Slug != "item-1" {
		t.Fatalf("first identifiers = %#v", first)
	}
	if first.Name != "AK test item" || first.Description != "Base description" {
		t.Fatalf("first item = %#v", first)
	}
	if first.ImageURL != "https://assets.tarkov.dev/item-1-512.webp" {
		t.Fatalf("first image url = %q", first.ImageURL)
	}
	if len(first.Translations) != 1 {
		t.Fatalf("first translations = %#v", first.Translations)
	}
	if first.Translations[0].LanguageCode != "ru" || first.Translations[0].Name != "АК тестовый предмет" || first.Translations[0].Description != "Русское описание" {
		t.Fatalf("first translation = %#v", first.Translations[0])
	}

	second := items[1]
	if second.ImageURL != "https://assets.tarkov.dev/item-2-icon.webp" {
		t.Fatalf("second image url = %q", second.ImageURL)
	}
	if len(second.Translations) != 1 || second.Translations[0].Name != "Второй предмет" {
		t.Fatalf("second translation = %#v", second.Translations)
	}
}
