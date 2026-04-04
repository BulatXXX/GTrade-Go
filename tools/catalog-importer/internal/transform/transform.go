package transform

import "gtrade/tools/catalog-importer/internal/source"

type Item struct {
	Game         string
	Source       string
	ExternalID   string
	Slug         string
	Name         string
	Description  string
	ImageURL     string
	Translations []Translation
}

type Translation struct {
	LanguageCode string `json:"language_code"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
}

type Transformer interface {
	Transform(raw []source.RawItem) ([]Item, error)
}

type NoopTransformer struct{}

func NewNoopTransformer() *NoopTransformer {
	return &NoopTransformer{}
}

func (t *NoopTransformer) Transform(raw []source.RawItem) ([]Item, error) {
	items := make([]Item, 0, len(raw))
	for _, r := range raw {
		translations := make([]Translation, 0, len(r.Translations))
		for _, tr := range r.Translations {
			translations = append(translations, Translation{
				LanguageCode: tr.LanguageCode,
				Name:         tr.Name,
				Description:  tr.Description,
			})
		}
		items = append(items, Item{
			Game:         r.Game,
			Source:       r.Source,
			ExternalID:   r.ExternalID,
			Slug:         r.Slug,
			Name:         r.Name,
			Description:  r.Description,
			ImageURL:     r.ImageURL,
			Translations: translations,
		})
	}
	return items, nil
}
