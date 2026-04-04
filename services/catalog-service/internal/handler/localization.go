package handler

import (
	"strings"

	"gtrade/services/catalog-service/internal/model"
)

func localizeItem(item model.Item, language string) model.Item {
	language = strings.TrimSpace(strings.ToLower(language))
	if language == "" {
		return item
	}

	item.LocalizedLanguage = language
	item.LocalizedName = item.Name
	item.LocalizedDescription = item.Description

	for _, translation := range item.Translations {
		if strings.EqualFold(strings.TrimSpace(translation.LanguageCode), language) {
			if strings.TrimSpace(translation.Name) != "" {
				item.LocalizedName = translation.Name
			}
			if strings.TrimSpace(translation.Description) != "" {
				item.LocalizedDescription = translation.Description
			}
			break
		}
	}

	return item
}

func localizeItems(items []model.Item, language string) []model.Item {
	if strings.TrimSpace(language) == "" {
		return items
	}

	out := make([]model.Item, 0, len(items))
	for _, item := range items {
		out = append(out, localizeItem(item, language))
	}
	return out
}
