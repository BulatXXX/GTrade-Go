package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	defaultEVEBaseURL   = "https://esi.evetech.net/latest"
	defaultEVEImageBase = "https://images.evetech.net/types"
)

type EVESource struct {
	client   *http.Client
	baseURL  string
	language string
	limit    int
}

func NewEVESource(client *http.Client, language string, limit int) *EVESource {
	if language == "" {
		language = "en"
	}
	return &EVESource{
		client:   client,
		baseURL:  defaultEVEBaseURL,
		language: strings.ToLower(language),
		limit:    limit,
	}
}

func (s *EVESource) Stream(ctx context.Context, consume func(RawItem) error) error {
	priceEntries, err := s.fetchMarketPrices(ctx)
	if err != nil {
		return err
	}

	processed := 0
	for _, entry := range priceEntries {
		baseType, err := s.fetchType(ctx, entry.TypeID, "en")
		if err != nil {
			if isSkippableEVETypeError(err) {
				continue
			}
			return err
		}
		if !baseType.Published {
			continue
		}

		externalID := strconv.Itoa(baseType.TypeID)
		raw := RawItem{
			Game:        "eve",
			Source:      "esi",
			ExternalID:  externalID,
			Slug:        buildEVESlug(baseType.Name, baseType.TypeID),
			Name:        baseType.Name,
			Description: strings.TrimSpace(baseType.Description),
			ImageURL:    resolveEVEImageURL(baseType.TypeID),
		}

		if s.language != "" && s.language != "en" {
			localizedType, err := s.fetchType(ctx, entry.TypeID, s.language)
			if err != nil {
				if !isSkippableEVETypeError(err) {
					return err
				}
			} else {
				localizedName := strings.TrimSpace(localizedType.Name)
				localizedDescription := strings.TrimSpace(localizedType.Description)
				if localizedName != "" || localizedDescription != "" {
					raw.Translations = []RawTranslation{{
						LanguageCode: s.language,
						Name:         fallbackString(localizedName, raw.Name),
						Description:  localizedDescription,
					}}
				}
			}
		}

		if err := consume(raw); err != nil {
			return err
		}

		processed++
		if s.limit > 0 && processed >= s.limit {
			break
		}

		time.Sleep(25 * time.Millisecond)
	}

	return nil
}

func (s *EVESource) fetchMarketPrices(ctx context.Context) ([]eveMarketPrice, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/markets/prices/?datasource=tranquility", nil)
	if err != nil {
		return nil, fmt.Errorf("build eve market prices request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request eve market prices: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("eve market prices request failed: status=%d", resp.StatusCode)
	}

	var payload []eveMarketPrice
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode eve market prices response: %w", err)
	}

	return payload, nil
}

func (s *EVESource) fetchType(ctx context.Context, typeID int, language string) (*eveType, error) {
	url := fmt.Sprintf("%s/universe/types/%d/?datasource=tranquility&language=%s", s.baseURL, typeID, language)

	var lastStatus int
	backoff := 500 * time.Millisecond
	for attempt := 0; attempt < 5; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("build eve type request for %d: %w", typeID, err)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := s.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request eve type %d: %w", typeID, err)
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

		if resp.StatusCode == http.StatusNotFound {
			return nil, errEVETypeNotFound
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("eve type request failed for %d: status=%d", typeID, resp.StatusCode)
		}

		var payload eveType
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return nil, fmt.Errorf("decode eve type response for %d: %w", typeID, err)
		}

		return &payload, nil
	}

	return nil, fmt.Errorf("eve type request failed for %d: status=%d after retries", typeID, lastStatus)
}

func resolveEVEImageURL(typeID int) string {
	return fmt.Sprintf("%s/%d/icon?size=128", defaultEVEImageBase, typeID)
}

func buildEVESlug(name string, typeID int) string {
	var b strings.Builder
	lastHyphen := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastHyphen = false
		case !lastHyphen:
			b.WriteByte('-')
			lastHyphen = true
		}
	}

	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return fmt.Sprintf("type-%d", typeID)
	}
	return fmt.Sprintf("%s-%d", slug, typeID)
}

func isSkippableEVETypeError(err error) bool {
	return err != nil && strings.Contains(err.Error(), errEVETypeNotFound.Error())
}

var errEVETypeNotFound = fmt.Errorf("eve type not found")

type eveMarketPrice struct {
	TypeID int `json:"type_id"`
}

type eveType struct {
	TypeID      int    `json:"type_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Published   bool   `json:"published"`
}
