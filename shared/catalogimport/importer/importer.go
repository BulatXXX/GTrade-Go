package importer

import (
	"context"

	"github.com/singularity/gtrade/shared/catalogimport/source"
	"github.com/singularity/gtrade/shared/catalogimport/transform"
)

type Repository interface {
	Upsert(ctx context.Context, item transform.Item) error
}

type Observer interface {
	OnStart(total int)
	OnItemProcessed(item transform.Item, processed, total int)
	OnFinish(processed, total int)
}

type Importer struct {
	source      source.Source
	transformer transform.Transformer
	repository  Repository
	observer    Observer
}

func New(src source.Source, tr transform.Transformer, repo Repository) *Importer {
	return &Importer{
		source:      src,
		transformer: tr,
		repository:  repo,
	}
}

func (i *Importer) WithObserver(observer Observer) *Importer {
	i.observer = observer
	return i
}

func (i *Importer) Run(ctx context.Context) (int, int, error) {
	total, err := i.source.TotalHint(ctx)
	if err != nil {
		return 0, 0, err
	}

	if i.observer != nil {
		i.observer.OnStart(total)
	}

	processed := 0
	err = i.source.Stream(ctx, func(raw source.RawItem) error {
		item, err := i.transformer.Transform(raw)
		if err != nil {
			return err
		}
		if err := i.repository.Upsert(ctx, item); err != nil {
			return err
		}

		processed++
		if i.observer != nil {
			i.observer.OnItemProcessed(item, processed, total)
		}
		return nil
	})
	if err != nil {
		return processed, total, err
	}

	if i.observer != nil {
		i.observer.OnFinish(processed, total)
	}

	return processed, total, nil
}
