package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gtrade/services/catalog-service/internal/model"
)

func (h *Handler) GetStats(c *gin.Context) {
	stats, err := h.catalogService.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get stats failed"})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *Handler) StartPriceHistorySync(c *gin.Context) {
	job := h.adminService.StartPriceHistorySync(context.WithoutCancel(c.Request.Context()))
	c.JSON(http.StatusAccepted, toAdminJobResponse(job))
}

func (h *Handler) StartCatalogImport(c *gin.Context) {
	var req model.AdminCatalogImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	job, err := h.adminService.StartCatalogImport(context.WithoutCancel(c.Request.Context()), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, toAdminJobResponse(job))
}

func (h *Handler) GetLocalizationCoverage(c *gin.Context) {
	coverage, err := h.catalogService.GetLocalizationCoverage(c.Request.Context(), c.Query("game"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get localization coverage failed"})
		return
	}
	c.JSON(http.StatusOK, coverage)
}

func (h *Handler) ListAdminJobs(c *gin.Context) {
	jobs := h.adminService.ListJobs()
	resp := make([]model.AdminJobStatusResponse, 0, len(jobs))
	for _, job := range jobs {
		resp = append(resp, toAdminJobResponse(job))
	}
	c.JSON(http.StatusOK, gin.H{"jobs": resp})
}

func (h *Handler) GetAdminJob(c *gin.Context) {
	job := h.adminService.GetJob(c.Param("id"))
	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	c.JSON(http.StatusOK, toAdminJobResponse(job))
}

func (h *Handler) ListSchedulerStates(c *gin.Context) {
	resp, err := h.adminService.ListSchedulerStates(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func toAdminJobResponse(job interface {
	GetID() string
}) model.AdminJobStatusResponse {
	typed := job.(interface {
		GetType() string
		GetStatus() string
		GetProgressPercent() int
		GetProcessed() int
		GetTotal() int
		GetError() string
		GetStartedAt() time.Time
		GetFinishedAt() *time.Time
		GetMeta() map[string]string
	})

	resp := model.AdminJobStatusResponse{
		ID:              job.GetID(),
		Type:            typed.GetType(),
		Status:          typed.GetStatus(),
		ProgressPercent: typed.GetProgressPercent(),
		Processed:       typed.GetProcessed(),
		Total:           typed.GetTotal(),
		Error:           typed.GetError(),
		StartedAt:       typed.GetStartedAt().Format(time.RFC3339),
		Meta:            typed.GetMeta(),
	}
	if finishedAt := typed.GetFinishedAt(); finishedAt != nil {
		resp.FinishedAt = finishedAt.Format(time.RFC3339)
	}
	return resp
}
