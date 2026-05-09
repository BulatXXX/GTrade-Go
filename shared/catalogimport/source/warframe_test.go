package source

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

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
							{"id":"1","slug":"frost_prime_set","i18n":{"en":{"name":"Frost Prime Set","icon":"items/images/en/frost_prime_set.png"}}},
							{"id":"2","slug":"abandoned_market_key","i18n":{"en":{"name":"Abandoned Key","icon":"items/images/en/abandoned.png"}}}
						]
					}`)),
				}, nil
			case "/orders/item/frost_prime_set/top":
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`{
						"data": {
							"sell": [{"platinum": 12}, {"platinum": 14}],
							"buy":  [{"platinum": 8}]
						}
					}`)),
				}, nil
			case "/orders/item/abandoned_market_key/top":
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`{
						"data": {"sell": [], "buy": []}
					}`)),
				}, nil
			case "/items/frost_prime_set":
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`{
						"data": {
							"id":"1",
							"slug":"frost_prime_set",
							"i18n":{"en":{"name":"Frost Prime Set","description":"Frost Prime warframe set","icon":"items/images/en/frost_prime_set.png"}}
						}
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
	if items[0].Slug != "frost_prime_set" {
		t.Fatalf("item = %#v", items[0])
	}
}

func TestWarframeSourceStream_SkipsOn404Orders(t *testing.T) {
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
							{"id":"1","slug":"missing_item","i18n":{"en":{"name":"Missing","icon":"x.png"}}}
						]
					}`)),
				}, nil
			case "/orders/item/missing_item/top":
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"error":"not_found"}`)),
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
	err := src.Stream(context.Background(), func(item RawItem) error {
		items = append(items, item)
		return nil
	})
	if err != nil {
		t.Fatalf("stream items: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("items len = %d, want 0", len(items))
	}
}
