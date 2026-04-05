package marketplace

import (
	"context"
	"net/http"
	"testing"

	"gtrade/services/api-integration-service/internal/model"
)

func TestEVEClientSearchItems_ReturnsEmptyBecauseCatalogOwnsSearch(t *testing.T) {
	t.Parallel()

	client := NewEVEClientWithBaseURL("https://esi.example.test", "https://images.example.test/types", &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			t.Fatalf("unexpected outbound request: %s", r.URL.String())
			return nil, nil
		}),
	})

	items, err := client.SearchItems(context.Background(), model.SearchItemsQuery{
		Game:  "eve",
		Query: "tritanium",
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("SearchItems: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0", len(items))
	}
}

func TestEVEClientGetItem_MapsTypeCard(t *testing.T) {
	t.Parallel()

	client := NewEVEClientWithBaseURL(
		"https://esi.example.test",
		"https://images.example.test/types",
		&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/universe/types/34/" {
				t.Fatalf("path = %s, want /universe/types/34/", r.URL.Path)
			}
			return jsonResponse(`{
				"name":"Tritanium",
				"description":"Refined mineral."
			}`), nil
		})},
	)

	item, err := client.GetItem(context.Background(), model.GetItemQuery{
		Game: "eve",
		ID:   "34",
	})
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}

	if item.Name != "Tritanium" {
		t.Fatalf("name = %q", item.Name)
	}
	if item.ImageURL != "https://images.example.test/types/34/icon?size=128" {
		t.Fatalf("image_url = %q", item.ImageURL)
	}
	if item.Currency != "ISK" {
		t.Fatalf("currency = %q", item.Currency)
	}
}

func TestEVEClientGetPricing_MapsMarketPrices(t *testing.T) {
	t.Parallel()

	client := NewEVEClientWithBaseURL(
		"https://esi.example.test",
		"https://images.example.test/types",
		&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/markets/prices/" {
				t.Fatalf("path = %s, want /markets/prices/", r.URL.Path)
			}
			return jsonResponse(`[
				{"type_id":18,"adjusted_price":34.4,"average_price":31.55},
				{"type_id":34,"adjusted_price":3.06,"average_price":3.9}
			]`), nil
		})},
	)

	price, err := client.GetPricing(context.Background(), model.GetPricingQuery{
		Game: "eve",
		ID:   "34",
	})
	if err != nil {
		t.Fatalf("GetPricing: %v", err)
	}

	if price.Pricing.Current == nil || *price.Pricing.Current != 3.9 {
		t.Fatalf("current = %#v", price.Pricing.Current)
	}
	if price.Pricing.AdjustedPrice == nil || *price.Pricing.AdjustedPrice != 3.06 {
		t.Fatalf("adjusted_price = %#v", price.Pricing.AdjustedPrice)
	}
	if price.MarketKind != "reference_market" {
		t.Fatalf("market_kind = %q", price.MarketKind)
	}
}
