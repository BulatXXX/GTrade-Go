package source

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

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
						{"type_id":36,"average_price":null,"adjusted_price":null},
						{"type_id":37,"adjusted_price":12.5}
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
			case "/universe/types/37/":
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`{
						"type_id":37,
						"name":"Mexallon",
						"description":"Plain text description.",
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
	if first.ExternalID != "34" {
		t.Fatalf("first external_id = %q", first.ExternalID)
	}
	wantDesc := "Refined mineral.\n\nSee Pyerite."
	if first.Description != wantDesc {
		t.Fatalf("first description = %q, want %q", first.Description, wantDesc)
	}

	second := items[1]
	if second.ExternalID != "37" {
		t.Fatalf("second external_id = %q", second.ExternalID)
	}
}

func TestEVESourceTotalHint_CountsOnlyPricedTypes(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/markets/prices/" {
				t.Fatalf("unexpected path: %s", req.URL.Path)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(`[
					{"type_id":1,"average_price":1.0},
					{"type_id":2},
					{"type_id":3,"adjusted_price":2.0}
				]`)),
			}, nil
		}),
	}

	src := NewEVESource(client, "en", 0)
	src.baseURL = "https://example.test"

	total, err := src.TotalHint(context.Background())
	if err != nil {
		t.Fatalf("TotalHint: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
