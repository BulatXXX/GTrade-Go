package transform

import "gtrade/tools/catalog-importer/internal/source"

type Item struct {
	ExternalID string
	Name       string
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
		items = append(items, Item{ExternalID: r.ID, Name: r.Name})
	}
	return items, nil
}
