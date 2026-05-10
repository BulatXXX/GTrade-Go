package repository

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/jackc/pgx/v5"
)

type SchedulerState struct {
	JobName        string
	Status         string
	LastStartedAt  *time.Time
	LastFinishedAt *time.Time
	LastError      *string
	LastProcessed  int
	LastTotal      int
	RunsTotal      int64
	UpdatedAt      time.Time
}

func SchedulerLockKey(jobName string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(jobName))
	return int64(h.Sum64())
}

// AcquireSchedulerLock tries to take a session-level advisory lock for the
// given job. Returns (true, release, nil) when the lock is granted; the
// returned release function must be called to free the lock and to release the
// underlying connection back to the pool. Returns (false, noop, nil) when the
// lock is already held.
func (r *UserAssetRepository) AcquireSchedulerLock(ctx context.Context, lockKey int64) (bool, func(), error) {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return false, func() {}, fmt.Errorf("acquire conn: %w", err)
	}

	var acquired bool
	if err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", lockKey).Scan(&acquired); err != nil {
		conn.Release()
		return false, func() {}, fmt.Errorf("pg_try_advisory_lock: %w", err)
	}
	if !acquired {
		conn.Release()
		return false, func() {}, nil
	}

	release := func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = conn.Exec(releaseCtx, "SELECT pg_advisory_unlock($1)", lockKey)
		conn.Release()
	}
	return true, release, nil
}

func (r *UserAssetRepository) MarkSchedulerStarted(ctx context.Context, jobName string, startedAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO schedule_state (job_name, status, last_started_at, last_error, updated_at)
		VALUES ($1, 'running', $2, NULL, NOW())
		ON CONFLICT (job_name) DO UPDATE SET
			status = EXCLUDED.status,
			last_started_at = EXCLUDED.last_started_at,
			last_error = NULL,
			updated_at = NOW()
	`, jobName, startedAt)
	if err != nil {
		return fmt.Errorf("mark scheduler started: %w", err)
	}
	return nil
}

func (r *UserAssetRepository) MarkSchedulerFinished(ctx context.Context, jobName string, finishedAt time.Time, runErr error, processed, total int) error {
	status := "ok"
	var errStr *string
	if runErr != nil {
		status = "error"
		s := runErr.Error()
		errStr = &s
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO schedule_state (job_name, status, last_finished_at, last_error, last_processed, last_total, runs_total, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 1, NOW())
		ON CONFLICT (job_name) DO UPDATE SET
			status = EXCLUDED.status,
			last_finished_at = EXCLUDED.last_finished_at,
			last_error = EXCLUDED.last_error,
			last_processed = EXCLUDED.last_processed,
			last_total = EXCLUDED.last_total,
			runs_total = schedule_state.runs_total + 1,
			updated_at = NOW()
	`, jobName, status, finishedAt, errStr, processed, total)
	if err != nil {
		return fmt.Errorf("mark scheduler finished: %w", err)
	}
	return nil
}

func (r *UserAssetRepository) ListSchedulerStates(ctx context.Context) ([]SchedulerState, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT job_name, status, last_started_at, last_finished_at, last_error, last_processed, last_total, runs_total, updated_at
		FROM schedule_state
		ORDER BY job_name
	`)
	if err != nil {
		return nil, fmt.Errorf("list scheduler states: %w", err)
	}
	defer rows.Close()

	out := make([]SchedulerState, 0)
	for rows.Next() {
		var s SchedulerState
		if err := rows.Scan(
			&s.JobName,
			&s.Status,
			&s.LastStartedAt,
			&s.LastFinishedAt,
			&s.LastError,
			&s.LastProcessed,
			&s.LastTotal,
			&s.RunsTotal,
			&s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan scheduler state: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("rows iterate: %w", err)
	}
	return out, nil
}
