package adminjobs

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/singularity/gtrade/shared/catalogimport/importer"
	"github.com/singularity/gtrade/shared/catalogimport/source"
	"github.com/singularity/gtrade/shared/catalogimport/transform"
	"gtrade/services/catalog-service/internal/model"
)

type CatalogImportUpserter interface {
	UpsertItem(ctx context.Context, input model.CreateItemInput) (*model.Item, error)
}

type CatalogImportRunner struct {
	manager *Manager
	service CatalogImportUpserter
}

func NewCatalogImportRunner(manager *Manager, service CatalogImportUpserter) *CatalogImportRunner {
	return &CatalogImportRunner{
		manager: manager,
		service: service,
	}
}

func (r *CatalogImportRunner) StartCatalogImport(ctx context.Context, req model.AdminCatalogImportRequest) (*Job, error) {
	game := strings.ToLower(strings.TrimSpace(req.Game))
	language := strings.ToLower(strings.TrimSpace(req.Language))
	if game == "" {
		return nil, fmt.Errorf("game is required")
	}
	if req.Limit < 0 {
		return nil, fmt.Errorf("limit must be >= 0")
	}

	src, err := source.New(source.Config{
		Name:       game,
		Language:   language,
		Limit:      req.Limit,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	})
	if err != nil {
		return nil, err
	}

	meta := map[string]string{
		"game": game,
	}
	if language != "" {
		meta["language"] = language
	}
	if req.Limit > 0 {
		meta["limit"] = fmt.Sprintf("%d", req.Limit)
	}

	repo := &catalogImportRepository{service: r.service}
	tr := transform.NewNoopTransformer()

	job := r.manager.StartWithMeta(ctx, "catalog-import", meta, func(ctx context.Context, job *Job) error {
		observer := &catalogImportObserver{manager: r.manager, jobID: job.ID}
		_, _, err := importer.New(src, tr, repo).WithObserver(observer).Run(ctx)
		return err
	})

	return job, nil
}

type catalogImportRepository struct {
	service CatalogImportUpserter
}

func (r *catalogImportRepository) Upsert(ctx context.Context, item transform.Item) error {
	_, err := r.service.UpsertItem(ctx, model.CreateItemInput{
		Game:         item.Game,
		Source:       item.Source,
		ExternalID:   item.ExternalID,
		Slug:         item.Slug,
		Name:         item.Name,
		Description:  item.Description,
		ImageURL:     item.ImageURL,
		Translations: toModelTranslations(item.Translations),
	})
	return err
}

func toModelTranslations(translations []transform.Translation) []model.ItemTranslation {
	if len(translations) == 0 {
		return nil
	}
	out := make([]model.ItemTranslation, 0, len(translations))
	for _, translation := range translations {
		out = append(out, model.ItemTranslation{
			LanguageCode: translation.LanguageCode,
			Name:         translation.Name,
			Description:  translation.Description,
		})
	}
	return out
}

type catalogImportObserver struct {
	manager *Manager
	jobID   string
}

func (o *catalogImportObserver) OnStart(total int) {
	o.manager.UpdateProgress(o.jobID, 0, total)
}

func (o *catalogImportObserver) OnItemProcessed(_ transform.Item, processed, total int) {
	o.manager.UpdateProgress(o.jobID, processed, total)
}

func (o *catalogImportObserver) OnFinish(processed, total int) {
	o.manager.UpdateProgress(o.jobID, processed, total)
}
