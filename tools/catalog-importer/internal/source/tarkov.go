package source

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const defaultTarkovBaseURL = "https://api.tarkov.dev/graphql"

type TarkovSource struct {
	client   *http.Client
	baseURL  string
	language string
	limit    int
}

func NewTarkovSource(client *http.Client, language string, limit int) *TarkovSource {
	if language == "" {
		language = "en"
	}
	return &TarkovSource{
		client:   client,
		baseURL:  defaultTarkovBaseURL,
		language: strings.ToLower(language),
		limit:    limit,
	}
}

func (s *TarkovSource) Stream(ctx context.Context, consume func(RawItem) error) error {
	const pageSize = 1000

	processed := 0
	offset := 0

	for {
		batchLimit := pageSize
		if s.limit > 0 {
			remaining := s.limit - processed
			if remaining <= 0 {
				break
			}
			if remaining < batchLimit {
				batchLimit = remaining
			}
		}

		baseItems, err := s.fetchItems(ctx, "en", offset, batchLimit)
		if err != nil {
			return err
		}
		if len(baseItems) == 0 {
			break
		}

		localizedByID := map[string]tarkovItem{}
		if s.language != "" && s.language != "en" {
			localizedItems, err := s.fetchItems(ctx, s.language, offset, batchLimit)
			if err != nil {
				return err
			}
			for _, item := range localizedItems {
				localizedByID[item.ID] = item
			}
		}

		for _, item := range baseItems {
			raw := RawItem{
				Game:        "tarkov",
				Source:      "tarkov-dev",
				ExternalID:  item.ID,
				Slug:        item.ID,
				Name:        strings.TrimSpace(item.Name),
				Description: strings.TrimSpace(item.Description),
				ImageURL:    fallbackString(strings.TrimSpace(item.Image512pxLink), strings.TrimSpace(item.IconLink)),
			}

			if localizedItem, ok := localizedByID[item.ID]; ok {
				localizedName := strings.TrimSpace(localizedItem.Name)
				localizedDescription := strings.TrimSpace(localizedItem.Description)
				if localizedName != "" || localizedDescription != "" {
					raw.Translations = []RawTranslation{{
						LanguageCode: s.language,
						Name:         fallbackString(localizedName, raw.Name),
						Description:  localizedDescription,
					}}
				}
			}

			if err := consume(raw); err != nil {
				return err
			}
			processed++
			if s.limit > 0 && processed >= s.limit {
				return nil
			}
		}

		if len(baseItems) < batchLimit {
			break
		}

		offset += len(baseItems)
		time.Sleep(50 * time.Millisecond)
	}

	return nil
}

func (s *TarkovSource) fetchItems(ctx context.Context, language string, offset, limit int) ([]tarkovItem, error) {
	query := fmt.Sprintf(`{ items(lang: %s, gameMode: regular, offset: %d, limit: %d) { id name description iconLink image512pxLink } }`, language, offset, limit)
	body, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		return nil, fmt.Errorf("marshal tarkov graphql query: %w", err)
	}

	var lastStatus int
	backoff := 500 * time.Millisecond
	for attempt := 0; attempt < 5; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("build tarkov items request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "GTrade catalog-importer")

		resp, err := s.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request tarkov items: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
			lastStatus = resp.StatusCode
			retryDelay := retryAfterDelay(resp.Header.Get("Retry-After"), backoff)
			resp.Body.Close()
			time.Sleep(retryDelay)
			backoff *= 2
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("tarkov items request failed: status=%d", resp.StatusCode)
		}

		var payload tarkovItemsResponse
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return nil, fmt.Errorf("decode tarkov items response: %w", err)
		}
		if len(payload.Errors) > 0 {
			return nil, fmt.Errorf("tarkov graphql returned errors")
		}

		return payload.Data.Items, nil
	}

	return nil, fmt.Errorf("tarkov items request failed: status=%d after retries", lastStatus)
}

type tarkovItemsResponse struct {
	Data struct {
		Items []tarkovItem `json:"items"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type tarkovItem struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	IconLink       string `json:"iconLink"`
	Image512pxLink string `json:"image512pxLink"`
}
