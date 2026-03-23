package importer

import (
	"gtrade/tools/catalog-importer/internal/repository"
	"gtrade/tools/catalog-importer/internal/source"
	"gtrade/tools/catalog-importer/internal/transform"
)

type Importer struct {
	source      source.Source
	transformer transform.Transformer
	repository  repository.Repository
}

func New(src source.Source, tr transform.Transformer, repo repository.Repository) *Importer {
	return &Importer{source: src, transformer: tr, repository: repo}
}

func (i *Importer) Run() error {
	raw, err := i.source.Fetch()
	if err != nil {
		return err
	}
	items, err := i.transformer.Transform(raw)
	if err != nil {
		return err
	}
	return i.repository.Upsert(items)
}
