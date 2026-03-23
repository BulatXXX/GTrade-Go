package repository

import "gtrade/tools/catalog-importer/internal/transform"

type Repository interface {
	Upsert(items []transform.Item) error
}

type NoopRepository struct{}

func NewNoopRepository() *NoopRepository {
	return &NoopRepository{}
}

func (r *NoopRepository) Upsert(_ []transform.Item) error {
	return nil
}
