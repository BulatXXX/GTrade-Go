package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gtrade/services/catalog-service/internal/adminjobs"
	"gtrade/services/catalog-service/internal/model"
)

type stubAdminUseCase struct {
	startCatalogImportFn func(ctx context.Context, req model.AdminCatalogImportRequest) (*adminjobs.Job, error)
	listJobsFn           func() []*adminjobs.Job
	getJobFn             func(id string) *adminjobs.Job
}

func (s stubAdminUseCase) StartPriceHistorySync(ctx context.Context) *adminjobs.Job {
	return &adminjobs.Job{ID: "job-1", Type: "price-history-sync", Status: "running", StartedAt: time.Now().UTC()}
}

func (s stubAdminUseCase) StartCatalogImport(ctx context.Context, req model.AdminCatalogImportRequest) (*adminjobs.Job, error) {
	return s.startCatalogImportFn(ctx, req)
}

func (s stubAdminUseCase) GetJob(id string) *adminjobs.Job {
	if s.getJobFn == nil {
		return nil
	}
	return s.getJobFn(id)
}

func (s stubAdminUseCase) ListJobs() []*adminjobs.Job {
	if s.listJobsFn == nil {
		return nil
	}
	return s.listJobsFn()
}

func TestStartCatalogImport_ReturnsAcceptedJob(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	var got model.AdminCatalogImportRequest
	h := New("catalog-service", stubCatalogUseCase{}, stubAdminUseCase{
		startCatalogImportFn: func(ctx context.Context, req model.AdminCatalogImportRequest) (*adminjobs.Job, error) {
			got = req
			return &adminjobs.Job{
				ID:              "job-1",
				Type:            "catalog-import",
				Status:          "running",
				ProgressPercent: 0,
				Processed:       0,
				Total:           100,
				StartedAt:       time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC),
				Meta:            map[string]string{"game": "warframe", "language": "ru"},
			}, nil
		},
	})

	router := gin.New()
	router.POST("/admin/jobs/catalog-import", h.StartCatalogImport)

	req := httptest.NewRequest(http.MethodPost, "/admin/jobs/catalog-import", strings.NewReader(`{"game":"warframe","language":"ru","limit":25}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	if got.Game != "warframe" || got.Language != "ru" || got.Limit != 25 {
		t.Fatalf("request = %#v", got)
	}

	var resp model.AdminJobStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Type != "catalog-import" || resp.Meta["game"] != "warframe" {
		t.Fatalf("response = %#v", resp)
	}
}

func TestGetLocalizationCoverage_ReturnsCoverage(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	h := New("catalog-service", stubCatalogUseCase{}, stubAdminUseCase{})
	h.catalogService = stubCatalogUseCaseWithCoverage{
		getLocalizationCoverageFn: func(ctx context.Context, game string) (*model.LocalizationCoverageResponse, error) {
			return &model.LocalizationCoverageResponse{
				Game: game,
				Coverage: []model.LocalizationCoverageRow{{
					Game:            "warframe",
					LanguageCode:    "ru",
					TotalItems:      10,
					TranslatedItems: 7,
					MissingItems:    3,
					CoveragePercent: 70,
				}},
			}, nil
		},
	}

	router := gin.New()
	router.GET("/admin/localizations/coverage", h.GetLocalizationCoverage)

	req := httptest.NewRequest(http.MethodGet, "/admin/localizations/coverage?game=warframe", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

type stubCatalogUseCaseWithCoverage struct {
	stubCatalogUseCase
	getLocalizationCoverageFn func(ctx context.Context, game string) (*model.LocalizationCoverageResponse, error)
}

func (s stubCatalogUseCaseWithCoverage) GetLocalizationCoverage(ctx context.Context, game string) (*model.LocalizationCoverageResponse, error) {
	return s.getLocalizationCoverageFn(ctx, game)
}
