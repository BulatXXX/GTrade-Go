package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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

func (s *WarframeSource) Fetch(ctx context.Context) ([]RawItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/items", nil)
	if err != nil {
		return nil, fmt.Errorf("build warframe items request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if s.language != "" {
		req.Header.Set("Language", s.language)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request warframe items: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("warframe items request failed: status=%d", resp.StatusCode)
	}

	var payload warframeItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode warframe items response: %w", err)
	}

	items := make([]RawItem, 0, len(payload.Data))
	for _, item := range payload.Data {
		name, imageURL := s.pickLocalizedFields(item)
		if strings.TrimSpace(item.Slug) == "" || strings.TrimSpace(name) == "" {
			continue
		}

		raw := RawItem{
			Game:       "warframe",
			Source:     "market",
			ExternalID: item.Slug,
			Slug:       item.Slug,
			Name:       name,
			ImageURL:   imageURL,
		}
		if s.language != "" && s.language != "en" {
			raw.Name = fallbackString(item.localizedName(), name)
			raw.Translations = []RawTranslation{{
				LanguageCode: s.language,
				Name:         raw.Name,
			}}
			raw.Name = fallbackString(item.englishName(), raw.Name)
		}
		items = append(items, raw)
		if s.limit > 0 && len(items) >= s.limit {
			break
		}
	}

	return items, nil
}

func (s *WarframeSource) pickLocalizedFields(item warframeItem) (string, string) {
	name := fallbackString(item.localizedName(), item.englishName(), item.Name, item.Slug)
	icon := fallbackString(item.localizedIcon(), item.englishIcon())
	if icon != "" {
		icon = resolveWarframeImageURL(icon)
	}
	return name, icon
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

type warframeItem struct {
	ID   string                       `json:"id"`
	Slug string                       `json:"slug"`
	Name string                       `json:"name"`
	I18n map[string]warframeLocalized `json:"i18n"`
}

type warframeLocalized struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

func (i warframeItem) englishName() string {
	return i.localizedField("en").Name
}

func (i warframeItem) localizedName() string {
	for lang, value := range i.I18n {
		if lang != "en" && strings.TrimSpace(value.Name) != "" {
			return value.Name
		}
	}
	return ""
}

func (i warframeItem) englishIcon() string {
	return i.localizedField("en").Icon
}

func (i warframeItem) localizedIcon() string {
	for lang, value := range i.I18n {
		if lang != "en" && strings.TrimSpace(value.Icon) != "" {
			return value.Icon
		}
	}
	return ""
}

func (i warframeItem) localizedField(language string) warframeLocalized {
	if i.I18n == nil {
		return warframeLocalized{}
	}
	return i.I18n[strings.ToLower(language)]
}
