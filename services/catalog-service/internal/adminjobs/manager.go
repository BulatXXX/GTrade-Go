package adminjobs

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Runner func(ctx context.Context, job *Job) error

type Job struct {
	ID              string
	Type            string
	Status          string
	ProgressPercent int
	Processed       int
	Total           int
	Error           string
	StartedAt       time.Time
	FinishedAt      *time.Time
	Meta            map[string]string
}

func (j *Job) GetID() string             { return j.ID }
func (j *Job) GetType() string           { return j.Type }
func (j *Job) GetStatus() string         { return j.Status }
func (j *Job) GetProgressPercent() int   { return j.ProgressPercent }
func (j *Job) GetProcessed() int         { return j.Processed }
func (j *Job) GetTotal() int             { return j.Total }
func (j *Job) GetError() string          { return j.Error }
func (j *Job) GetStartedAt() time.Time   { return j.StartedAt }
func (j *Job) GetFinishedAt() *time.Time { return j.FinishedAt }
func (j *Job) GetMeta() map[string]string {
	if len(j.Meta) == 0 {
		return nil
	}
	meta := make(map[string]string, len(j.Meta))
	for key, value := range j.Meta {
		meta[key] = value
	}
	return meta
}

type Manager struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

func NewManager() *Manager {
	return &Manager{jobs: map[string]*Job{}}
}

func (m *Manager) Start(ctx context.Context, jobType string, runner Runner) *Job {
	return m.StartWithMeta(ctx, jobType, nil, runner)
}

func (m *Manager) StartWithMeta(ctx context.Context, jobType string, meta map[string]string, runner Runner) *Job {
	job := &Job{
		ID:        fmt.Sprintf("%s-%d", jobType, time.Now().UnixNano()),
		Type:      jobType,
		Status:    "running",
		StartedAt: time.Now().UTC(),
		Meta:      cloneMeta(meta),
	}

	m.mu.Lock()
	m.jobs[job.ID] = job
	m.mu.Unlock()

	go func() {
		err := runner(ctx, job)
		m.mu.Lock()
		defer m.mu.Unlock()
		finishedAt := time.Now().UTC()
		job.FinishedAt = &finishedAt
		if err != nil {
			job.Status = "failed"
			job.Error = err.Error()
			return
		}
		job.Status = "completed"
		job.ProgressPercent = 100
	}()

	return cloneJob(job)
}

func (m *Manager) Get(id string) *Job {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, ok := m.jobs[id]
	if !ok {
		return nil
	}
	return cloneJob(job)
}

func (m *Manager) List() []*Job {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Job, 0, len(m.jobs))
	for _, job := range m.jobs {
		out = append(out, cloneJob(job))
	}
	return out
}

func (m *Manager) UpdateProgress(jobID string, processed, total int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[jobID]
	if !ok {
		return
	}
	job.Processed = processed
	job.Total = total
	if total > 0 {
		job.ProgressPercent = processed * 100 / total
	}
}

func cloneJob(job *Job) *Job {
	if job == nil {
		return nil
	}
	copy := *job
	copy.Meta = cloneMeta(job.Meta)
	return &copy
}

func cloneMeta(meta map[string]string) map[string]string {
	if len(meta) == 0 {
		return nil
	}
	copy := make(map[string]string, len(meta))
	for key, value := range meta {
		copy[key] = value
	}
	return copy
}
