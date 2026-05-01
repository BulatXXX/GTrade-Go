package marketplace

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"gtrade/services/api-integration-service/internal/model"
)

func TestTarkovClientSearchItems_UsesGameModeAndMapsResponse(t *testing.T) {
	t.Parallel()

	client := NewTarkovClientWithBaseURL("https://api.example.test/graphql", &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(r.Body)
			text := string(body)
			if !strings.Contains(text, `"gameMode":"pve"`) {
				t.Fatalf("request body missing pve gameMode: %s", text)
			}
			return jsonResponse(`{
				"data":{
					"items":[
						{
							"id":"5448bd6b4bdc2dfc2f8b4569",
							"name":"Makarov PM 9x18PM pistol",
							"types":["gun","wearable"],
							"description":"Pistol",
							"avg24hPrice":14161,
							"low24hPrice":5000,
							"high24hPrice":50000,
							"basePrice":4097,
							"changeLast48hPercent":-43.17,
							"width":2,
							"height":1,
							"iconLink":"https://assets.tarkov.dev/icon.webp",
							"image512pxLink":"https://assets.tarkov.dev/image.webp",
							"link":"https://tarkov.dev/item/makarov",
							"sellFor":[{"price":6969,"source":"fleaMarket","currency":"RUB"}]
						}
					]
				}
			}`), nil
		}),
	})

	items, err := client.SearchItems(context.Background(), model.SearchItemsQuery{
		Game:     "tarkov",
		GameMode: "pve",
		Query:    "Makarov",
		Limit:    5,
	})
	if err != nil {
		t.Fatalf("SearchItems: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].GameMode != "pve" {
		t.Fatalf("game_mode = %q", items[0].GameMode)
	}
	if items[0].ImageURL != "https://assets.tarkov.dev/image.webp" {
		t.Fatalf("image_url = %q", items[0].ImageURL)
	}
}

func TestTarkovClientGetItem_MapsItemResponse(t *testing.T) {
	t.Parallel()

	client := NewTarkovClientWithBaseURL("https://api.example.test/graphql", &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(r.Body)
			text := string(body)
			if !strings.Contains(text, "query ItemByID($id: ID!") {
				t.Fatalf("request body missing ID variable type: %s", text)
			}
			if !strings.Contains(text, `"gameMode":"regular"`) {
				t.Fatalf("request body missing regular gameMode: %s", text)
			}
			return jsonResponse(`{
				"data":{
					"item":{
						"id":"5448bd6b4bdc2dfc2f8b4569",
						"name":"Makarov PM 9x18PM pistol",
						"types":["gun"],
						"description":"Pistol",
						"iconLink":"https://assets.tarkov.dev/icon.webp",
						"image512pxLink":"https://assets.tarkov.dev/image.webp",
						"link":"https://tarkov.dev/item/makarov"
					}
				}
			}`), nil
		}),
	})

	item, err := client.GetItem(context.Background(), model.GetItemQuery{
		Game:     "tarkov",
		GameMode: "regular",
		ID:       "5448bd6b4bdc2dfc2f8b4569",
	})
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}
	if item.GameMode != "regular" {
		t.Fatalf("game_mode = %q", item.GameMode)
	}
	if item.URL != "https://tarkov.dev/item/makarov" {
		t.Fatalf("url = %q", item.URL)
	}
}

func TestTarkovClientGetItem_ReturnsUpstreamGraphQLError(t *testing.T) {
	t.Parallel()

	client := NewTarkovClientWithBaseURL("https://api.example.test/graphql", &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return jsonResponse(`{
				"errors":[
					{"message":"Variable \"$id\" of type \"String!\" used in position expecting type \"ID\"."}
				]
			}`), nil
		}),
	})

	_, err := client.GetItem(context.Background(), model.GetItemQuery{
		Game:     "tarkov",
		GameMode: "regular",
		ID:       "5448bd6b4bdc2dfc2f8b4569",
	})
	if err == nil {
		t.Fatal("GetItem error = nil, want graphql error")
	}
	if errors.Is(err, ErrNotFound) {
		t.Fatalf("GetItem error = %v, should not be translated to not found", err)
	}
	if !strings.Contains(err.Error(), "expecting type \"ID\"") {
		t.Fatalf("GetItem error = %v", err)
	}
}

func TestTarkovClientGetPricing_MapsAggregatedPrices(t *testing.T) {
	t.Parallel()

	client := NewTarkovClientWithBaseURL("https://api.example.test/graphql", &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(r.Body)
			text := string(body)
			if !strings.Contains(text, `"gameMode":"pve"`) {
				t.Fatalf("request body missing pve gameMode: %s", text)
			}
			return jsonResponse(`{
				"data":{
					"item":{
						"id":"5448bd6b4bdc2dfc2f8b4569",
						"name":"Makarov PM 9x18PM pistol",
						"avg24hPrice":14161,
						"low24hPrice":5000,
						"high24hPrice":50000,
						"basePrice":4097,
						"changeLast48hPercent":-43.17,
						"sellFor":[
							{"price":2048,"source":"prapor","currency":"RUB"},
							{"price":15,"source":"peacekeeper","currency":"USD"},
							{"price":6969,"source":"fleaMarket","currency":"RUB"}
						]
					}
				}
			}`), nil
		}),
	})

	price, err := client.GetPricing(context.Background(), model.GetPricingQuery{
		Game:     "tarkov",
		GameMode: "pve",
		ID:       "5448bd6b4bdc2dfc2f8b4569",
	})
	if err != nil {
		t.Fatalf("GetPricing: %v", err)
	}
	if price.GameMode != "pve" {
		t.Fatalf("game_mode = %q", price.GameMode)
	}
	if price.Pricing.Current == nil || *price.Pricing.Current != 14161 {
		t.Fatalf("current = %#v", price.Pricing.Current)
	}
	if price.Pricing.TopSell == nil || *price.Pricing.TopSell != 6969 {
		t.Fatalf("top_sell = %#v", price.Pricing.TopSell)
	}
	bestTrader, ok := price.RawContext["best_trader_source"]
	if !ok || bestTrader != "prapor" {
		t.Fatalf("best_trader_source = %#v", bestTrader)
	}
}
