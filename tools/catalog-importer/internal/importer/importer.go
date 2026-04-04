package importer

import (
	"context"
	"fmt"
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

func (i *Importer) Run(ctx context.Context) error {
	processed := 0
	return i.source.Stream(ctx, func(raw source.RawItem) error {
		item, err := i.transformer.Transform(raw)
		if err != nil {
			return err
		}
		if err := i.repository.Upsert([]transform.Item{item}); err != nil {
			return err
		}

		processed++
		if processed == 1 || processed%25 == 0 {
			fmt.Printf("catalog import progress: processed=%d game=%s source=%s slug=%s\n", processed, item.Game, item.Source, item.Slug)
		}

		return nil
	})
}
