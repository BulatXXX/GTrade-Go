package handler

import (
	"testing"

	"gtrade/services/catalog-service/internal/model"
)

func TestLocalizeItem_UsesTranslationWhenAvailable(t *testing.T) {
	t.Parallel()

	item := localizeItem(model.Item{
		Name:        "Frost Prime Set",
		Description: "Base description",
		Translations: []model.ItemTranslation{
			{LanguageCode: "ru", Name: "Набор Фроста Прайм", Description: "Перевод"},
		},
	}, "ru")

	if item.LocalizedName != "Набор Фроста Прайм" {
		t.Fatalf("localized name = %q", item.LocalizedName)
	}
	if item.LocalizedDescription != "Перевод" {
		t.Fatalf("localized description = %q", item.LocalizedDescription)
	}
	if item.LocalizedLanguage != "ru" {
		t.Fatalf("localized language = %q", item.LocalizedLanguage)
	}
}

func TestLocalizeItem_FallsBackToBaseFields(t *testing.T) {
	t.Parallel()

	item := localizeItem(model.Item{
		Name:        "Frost Prime Set",
		Description: "Base description",
	}, "de")

	if item.LocalizedName != "Frost Prime Set" {
		t.Fatalf("localized name = %q", item.LocalizedName)
	}
	if item.LocalizedDescription != "Base description" {
		t.Fatalf("localized description = %q", item.LocalizedDescription)
	}
	if item.LocalizedLanguage != "de" {
		t.Fatalf("localized language = %q", item.LocalizedLanguage)
	}
}
