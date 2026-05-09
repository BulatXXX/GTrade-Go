package source

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestTarkovSourceStream_SkipsItemsWithoutPrice(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			payload := string(body)

			if !strings.Contains(payload, "avg24hPrice") || !strings.Contains(payload, "basePrice") {
				t.Fatalf("graphql query missing price fields: %s", payload)
			}

			switch {
			case strings.Contains(payload, "lang: en") && strings.Contains(payload, "offset: 0"):
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`{
						"data": {
							"items": [
								{
									"id": "priced-item",
									"name": "Priced item",
									"description": "Has prices",
									"iconLink": "https://assets.tarkov.dev/priced-icon.webp",
									"image512pxLink": "https://assets.tarkov.dev/priced-512.webp",
									"avg24hPrice": 12500,
									"basePrice": 9000,
									"lastLowPrice": 11000
								},
								{
									"id": "no-price",
									"name": "Quest item",
									"description": "Untradeable",
									"iconLink": "https://assets.tarkov.dev/quest-icon.webp",
									"image512pxLink": "",
									"avg24hPrice": null,
									"basePrice": null,
									"lastLowPrice": null
								},
								{
									"id": "zero-price",
									"name": "Zero price",
									"description": "Suspect",
									"iconLink": "https://assets.tarkov.dev/zero-icon.webp",
									"image512pxLink": "",
									"avg24hPrice": 0,
									"basePrice": 0,
									"lastLowPrice": 0
								},
								{
									"id": "base-only",
									"name": "Base price only",
									"description": "Has base price",
									"iconLink": "https://assets.tarkov.dev/base-icon.webp",
									"image512pxLink": "",
									"basePrice": 500
								}
							]
						}
					}`)),
				}, nil
			case strings.Contains(payload, "offset: 4"):
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

	src := NewTarkovSource(client, "en", 0)
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
		t.Fatalf("items len = %d, want 2 (priced-item and base-only)", len(items))
	}

	if items[0].ExternalID != "priced-item" {
		t.Fatalf("items[0] = %#v", items[0])
	}
	if items[1].ExternalID != "base-only" {
		t.Fatalf("items[1] = %#v", items[1])
	}
}
