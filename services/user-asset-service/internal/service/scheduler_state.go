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
	repo      schedulerStateRepo
	schedules map[string]time.Duration
}

func NewSchedulerStateService(repo schedulerStateRepo, schedules map[string]time.Duration) *SchedulerStateService {
	return &SchedulerStateService{repo: repo, schedules: schedules}
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
		var intervalSeconds *int64
		var nextRunAt *string
		if interval, ok := s.schedules[st.JobName]; ok && interval > 0 {
			seconds := int64(interval / time.Second)
			intervalSeconds = &seconds
			if st.LastStartedAt != nil {
				next := st.LastStartedAt.Add(interval).UTC().Format(timeFormatRFC3339)
				nextRunAt = &next
			}
		}

		out = append(out, model.SchedulerStateItem{
			JobName:         st.JobName,
			Status:          st.Status,
			LastStartedAt:   formatNullableTime(st.LastStartedAt),
			LastFinishedAt:  formatNullableTime(st.LastFinishedAt),
			LastError:       st.LastError,
			LastProcessed:   st.LastProcessed,
			LastTotal:       st.LastTotal,
			RunsTotal:       st.RunsTotal,
			UpdatedAt:       st.UpdatedAt.UTC().Format(timeFormatRFC3339),
			IntervalSeconds: intervalSeconds,
			NextRunAt:       nextRunAt,
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
