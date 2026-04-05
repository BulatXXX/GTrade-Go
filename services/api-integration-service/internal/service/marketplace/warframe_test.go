package marketplace

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"gtrade/services/api-integration-service/internal/model"
)

func TestWarframeClientSearchItems_FiltersAndMapsResponse(t *testing.T) {
	t.Parallel()

	client := NewWarframeClientWithBaseURL(
		"https://api.example.test",
		"https://warframe.market/static/assets",
		"https://warframe.market/items",
		&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/items" {
				t.Fatalf("path = %s, want /items", r.URL.Path)
			}
			return jsonResponse(`{
			"data":[
				{"slug":"frost_prime_set","tags":["set","prime"],"i18n":{"en":{"name":"Frost Prime Set","icon":"items/images/en/frost_prime_set.png"}}},
				{"slug":"ember_prime_set","tags":["set","prime"],"i18n":{"en":{"name":"Ember Prime Set","icon":"items/images/en/ember_prime_set.png"}}}
			]
		}`), nil
		})},
	)

	items, err := client.SearchItems(context.Background(), model.SearchItemsQuery{
		Game:  "warframe",
		Query: "frost",
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("SearchItems: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ID != "frost_prime_set" {
		t.Fatalf("item id = %q", items[0].ID)
	}
	if items[0].ImageURL != "https://warframe.market/static/assets/items/images/en/frost_prime_set.png" {
		t.Fatalf("image_url = %q", items[0].ImageURL)
	}
}

func TestWarframeClientGetItem_MapsItemCard(t *testing.T) {
	t.Parallel()

	client := NewWarframeClientWithBaseURL(
		"https://api.example.test",
		"https://warframe.market/static/assets",
		"https://warframe.market/items",
		&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/items/frost_prime_set" {
				t.Fatalf("path = %s, want /items/frost_prime_set", r.URL.Path)
			}
			return jsonResponse(`{
			"data":{
				"slug":"frost_prime_set",
				"tags":["set","prime","warframe"],
				"i18n":{"en":{
					"name":"Frost Prime Set",
					"description":"Prime warframe set.",
					"icon":"items/images/en/frost_prime_set.png"
				}}
			}
		}`), nil
		})},
	)

	item, err := client.GetItem(context.Background(), model.GetItemQuery{
		Game: "warframe",
		ID:   "frost_prime_set",
	})
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}

	if item.Name != "Frost Prime Set" {
		t.Fatalf("name = %q", item.Name)
	}
	if item.URL != "https://warframe.market/items/frost_prime_set" {
		t.Fatalf("url = %q", item.URL)
	}
}

func TestWarframeClientGetPricing_MapsTopOrders(t *testing.T) {
	t.Parallel()

	client := NewWarframeClientWithBaseURL(
		"https://api.example.test",
		"https://warframe.market/static/assets",
		"https://warframe.market/items",
		&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/orders/item/frost_prime_set/top" {
				t.Fatalf("path = %s, want /orders/item/frost_prime_set/top", r.URL.Path)
			}
			return jsonResponse(`{
			"data":{
				"sell":[{"platinum":71},{"platinum":73},{"platinum":75}],
				"buy":[{"platinum":65},{"platinum":62}]
			}
		}`), nil
		})},
	)

	price, err := client.GetPricing(context.Background(), model.GetPricingQuery{
		Game: "warframe",
		ID:   "frost_prime_set",
	})
	if err != nil {
		t.Fatalf("GetPricing: %v", err)
	}

	if price.Pricing.TopSell == nil || *price.Pricing.TopSell != 71 {
		t.Fatalf("top_sell = %#v", price.Pricing.TopSell)
	}
	if price.Pricing.TopBuy == nil || *price.Pricing.TopBuy != 65 {
		t.Fatalf("top_buy = %#v", price.Pricing.TopBuy)
	}
	if price.Pricing.Spread == nil || *price.Pricing.Spread != 6 {
		t.Fatalf("spread = %#v", price.Pricing.Spread)
	}
	if price.Analytics.SampleSize == nil || *price.Analytics.SampleSize != 5 {
		t.Fatalf("sample_size = %#v", price.Analytics.SampleSize)
	}
	if price.Analytics.Low == nil || *price.Analytics.Low != 71 {
		t.Fatalf("low = %#v", price.Analytics.Low)
	}
	if price.Analytics.High == nil || *price.Analytics.High != 75 {
		t.Fatalf("high = %#v", price.Analytics.High)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
