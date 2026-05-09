package source

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestEVESourceStream_ParsesPublishedTypesAndLocalizedTranslation(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/markets/prices/":
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`[
						{"type_id":34,"average_price":3.9,"adjusted_price":3.06},
						{"type_id":35,"average_price":4.1,"adjusted_price":4.0}
					]`)),
				}, nil
			case "/universe/types/34/":
				if req.URL.Query().Get("language") == "ru" {
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     make(http.Header),
						Body: io.NopCloser(strings.NewReader(`{
							"type_id":34,
							"name":"Тританий",
							"description":"Русское описание",
							"published":true
						}`)),
					}, nil
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`{
						"type_id":34,
						"name":"Tritanium",
						"description":"Base description",
						"published":true
					}`)),
				}, nil
			case "/universe/types/35/":
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`{
						"type_id":35,
						"name":"Pyerite",
						"description":"Should be skipped",
						"published":false
					}`)),
				}, nil
			default:
				t.Fatalf("unexpected path: %s", req.URL.Path)
			}
			return nil, nil
		}),
	}

	src := NewEVESource(client, "ru", 10)
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
	if item.Game != "eve" || item.Source != "esi" {
		t.Fatalf("item = %#v", item)
	}
	if item.ExternalID != "34" {
		t.Fatalf("external_id = %q", item.ExternalID)
	}
	if item.Slug != "tritanium-34" {
		t.Fatalf("slug = %q", item.Slug)
	}
	if item.Name != "Tritanium" {
		t.Fatalf("name = %q", item.Name)
	}
	if item.Description != "Base description" {
		t.Fatalf("description = %q", item.Description)
	}
	if item.ImageURL != "https://images.evetech.net/types/34/icon?size=128" {
		t.Fatalf("image url = %q", item.ImageURL)
	}
	if len(item.Translations) != 1 {
		t.Fatalf("translations = %#v", item.Translations)
	}
	if item.Translations[0].LanguageCode != "ru" {
		t.Fatalf("translation = %#v", item.Translations[0])
	}
	if item.Translations[0].Name != "Тританий" || item.Translations[0].Description != "Русское описание" {
		t.Fatalf("translation = %#v", item.Translations[0])
	}
}

func TestEVESourceStream_SkipsItemsWithoutPriceAndStripsHTML(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/markets/prices/":
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`[
						{"type_id":34,"average_price":3.9,"adjusted_price":3.06},
						{"type_id":35},
						{"type_id":36,"average_price":null,"adjusted_price":null}
					]`)),
				}, nil
			case "/universe/types/34/":
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`{
						"type_id":34,
						"name":"Tritanium",
						"description":"<p>Refined <b>mineral</b>.</p><p>See <a href=\"showinfo:35\">Pyerite</a>.</p>",
						"published":true
					}`)),
				}, nil
			default:
				t.Fatalf("unexpected path: %s", req.URL.Path)
			}
			return nil, nil
		}),
	}

	src := NewEVESource(client, "en", 10)
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
	if items[0].ExternalID != "34" {
		t.Fatalf("external_id = %q", items[0].ExternalID)
	}
	wantDesc := "Refined mineral.\n\nSee Pyerite."
	if items[0].Description != wantDesc {
		t.Fatalf("description = %q, want %q", items[0].Description, wantDesc)
	}
}

func TestBuildEVESlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		typeID int
		want   string
	}{
		{name: "Tritanium", typeID: 34, want: "tritanium-34"},
		{name: "Large Shield Extender II", typeID: 1, want: "large-shield-extender-ii-1"},
		{name: "", typeID: 99, want: "type-99"},
	}

	for _, tt := range tests {
		if got := buildEVESlug(tt.name, tt.typeID); got != tt.want {
			t.Fatalf("buildEVESlug(%q, %d) = %q, want %q", tt.name, tt.typeID, got, tt.want)
		}
	}
}
