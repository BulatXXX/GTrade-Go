package service

import (
	"context"
	"fmt"
	"time"

	"gtrade/services/user-asset-service/internal/model"
	"gtrade/services/user-asset-service/internal/repository"
)

const timeFormatRFC3339 = "2006-01-02T15:04:05Z07:00"

type schedulerStateRepo interface {
	ListSchedulerStates(ctx context.Context) ([]repository.SchedulerState, error)
}

type SchedulerStateService struct {
	repo schedulerStateRepo
}

func NewSchedulerStateService(repo schedulerStateRepo) *SchedulerStateService {
	return &SchedulerStateService{repo: repo}
}

func (s *SchedulerStateService) List(ctx context.Context) (*model.SchedulerStateResponse, error) {
	if s == nil || s.repo == nil {
		return &model.SchedulerStateResponse{Items: []model.SchedulerStateItem{}}, nil
	}
	states, err := s.repo.ListSchedulerStates(ctx)
	if err != nil {
		return nil, fmt.Errorf("list scheduler states: %w", err)
	}
	out := make([]model.SchedulerStateItem, 0, len(states))
	for _, st := range states {
		out = append(out, model.SchedulerStateItem{
			JobName:        st.JobName,
			Status:         st.Status,
			LastStartedAt:  formatNullableTime(st.LastStartedAt),
			LastFinishedAt: formatNullableTime(st.LastFinishedAt),
			LastError:      st.LastError,
			LastProcessed:  st.LastProcessed,
			LastTotal:      st.LastTotal,
			RunsTotal:      st.RunsTotal,
			UpdatedAt:      st.UpdatedAt.UTC().Format(timeFormatRFC3339),
		})
	}
	return &model.SchedulerStateResponse{Items: out}, nil
}

func formatNullableTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	formatted := t.UTC().Format(timeFormatRFC3339)
	return &formatted
}
