package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultWarframeBaseURL   = "https://api.warframe.market/v2"
	defaultWarframeImageBase = "https://warframe.market/static/assets/"
)

type WarframeSource struct {
	client   *http.Client
	baseURL  string
	language string
	limit    int
}

func NewWarframeSource(client *http.Client, language string, limit int) *WarframeSource {
	if language == "" {
		language = "en"
	}
	return &WarframeSource{
		client:   client,
		baseURL:  defaultWarframeBaseURL,
		language: strings.ToLower(language),
		limit:    limit,
	}
}

func (s *WarframeSource) Stream(ctx context.Context, consume func(RawItem) error) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/items", nil)
	if err != nil {
		return fmt.Errorf("build warframe items request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Language", "en")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("request warframe items: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("warframe items request failed: status=%d", resp.StatusCode)
	}

	var payload warframeItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return fmt.Errorf("decode warframe items response: %w", err)
	}

	processed := 0
	for _, item := range payload.Data {
		if strings.TrimSpace(item.Slug) == "" {
			continue
		}

		baseItem, err := s.fetchItemDetail(ctx, item.Slug, "en")
		if err != nil {
			return err
		}

		raw := RawItem{
			Game:        "warframe",
			Source:      "market",
			ExternalID:  item.Slug,
			Slug:        item.Slug,
			Name:        fallbackString(baseItem.englishName(), item.englishName(), item.Name, item.Slug),
			Description: baseItem.englishDescription(),
			ImageURL:    resolveWarframeImageURL(fallbackString(baseItem.englishIcon(), item.englishIcon())),
		}

		if s.language != "" && s.language != "en" {
			localizedItem, err := s.fetchItemDetail(ctx, item.Slug, s.language)
			if err != nil {
				return err
			}
			localizedName := fallbackString(localizedItem.localizedName(s.language))
			localizedDescription := fallbackString(localizedItem.localizedDescription(s.language))
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
			break
		}

		time.Sleep(75 * time.Millisecond)
	}

	return nil
}

func (s *WarframeSource) fetchItemDetail(ctx context.Context, slug, language string) (*warframeItem, error) {
	var lastStatus int
	backoff := 500 * time.Millisecond

	for attempt := 0; attempt < 5; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/items/"+slug, nil)
		if err != nil {
			return nil, fmt.Errorf("build warframe item request for %s: %w", slug, err)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Language", language)

		resp, err := s.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request warframe item %s: %w", slug, err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			lastStatus = resp.StatusCode
			retryDelay := retryAfterDelay(resp.Header.Get("Retry-After"), backoff)
			resp.Body.Close()
			time.Sleep(retryDelay)
			backoff *= 2
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("warframe item request failed for %s: status=%d", slug, resp.StatusCode)
		}

		var payload warframeItemResponse
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return nil, fmt.Errorf("decode warframe item response for %s: %w", slug, err)
		}

		return &payload.Data, nil
	}

	return nil, fmt.Errorf("warframe item request failed for %s: status=%d after retries", slug, lastStatus)
}

func resolveWarframeImageURL(icon string) string {
	if strings.HasPrefix(icon, "http://") || strings.HasPrefix(icon, "https://") {
		return icon
	}
	return strings.TrimRight(defaultWarframeImageBase, "/") + "/" + strings.TrimLeft(icon, "/")
}

func fallbackString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

type warframeItemsResponse struct {
	Data []warframeItem `json:"data"`
}

type warframeItemResponse struct {
	Data warframeItem `json:"data"`
}

type warframeItem struct {
	ID   string                       `json:"id"`
	Slug string                       `json:"slug"`
	Name string                       `json:"name"`
	I18n map[string]warframeLocalized `json:"i18n"`
}

type warframeLocalized struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

func (i warframeItem) englishName() string {
	return i.localizedField("en").Name
}

func (i warframeItem) englishDescription() string {
	return i.localizedField("en").Description
}

func (i warframeItem) englishIcon() string {
	return i.localizedField("en").Icon
}

func (i warframeItem) localizedName(language string) string {
	return i.localizedField(language).Name
}

func (i warframeItem) localizedDescription(language string) string {
	return i.localizedField(language).Description
}

func (i warframeItem) localizedIcon(language string) string {
	return i.localizedField(language).Icon
}

func (i warframeItem) localizedField(language string) warframeLocalized {
	if i.I18n == nil {
		return warframeLocalized{}
	}
	return i.I18n[strings.ToLower(language)]
}

func retryAfterDelay(headerValue string, fallback time.Duration) time.Duration {
	if headerValue == "" {
		return fallback
	}
	seconds, err := strconv.Atoi(strings.TrimSpace(headerValue))
	if err != nil || seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}
