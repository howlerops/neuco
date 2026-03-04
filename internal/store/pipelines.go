package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/neuco-ai/neuco/internal/domain"
)

// CreatePipelineRun inserts a new pipeline run record.
func (s *Store) CreatePipelineRun(ctx context.Context, projectID uuid.UUID, pipelineType domain.PipelineType, metadata interface{}) (*domain.PipelineRun, error) {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		metadataJSON = []byte(`{}`)
	}

	const q = `
		INSERT INTO pipeline_runs (id, project_id, type, status, metadata)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, project_id, type, status, metadata, started_at, completed_at, error, created_at`

	id := uuid.New()
	var run domain.PipelineRun
	err = s.pool.QueryRow(ctx, q, id, projectID, pipelineType, domain.PipelineRunStatusPending, metadataJSON).Scan(
		&run.ID, &run.ProjectID, &run.Type, &run.Status,
		&run.Metadata, &run.StartedAt, &run.CompletedAt, &run.Error, &run.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("store.CreatePipelineRun: %w", err)
	}
	return &run, nil
}

// CreatePipelineTask inserts a task associated with a pipeline run.
func (s *Store) CreatePipelineTask(ctx context.Context, runID uuid.UUID, name string, sortOrder int) (*domain.PipelineTask, error) {
	const q = `
		INSERT INTO pipeline_tasks (id, pipeline_run_id, name, status, sort_order)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, pipeline_run_id, name, status, attempt, started_at, completed_at, duration_ms, error, sort_order`

	id := uuid.New()
	var task domain.PipelineTask
	err := s.pool.QueryRow(ctx, q, id, runID, name, domain.PipelineTaskStatusPending, sortOrder).Scan(
		&task.ID, &task.PipelineRunID, &task.Name, &task.Status,
		&task.Attempt, &task.StartedAt, &task.CompletedAt, &task.DurationMs, &task.Error, &task.SortOrder,
	)
	if err != nil {
		return nil, fmt.Errorf("store.CreatePipelineTask: %w", err)
	}
	return &task, nil
}

// UpdatePipelineTaskStatus advances a task's status and records timing info.
func (s *Store) UpdatePipelineTaskStatus(ctx context.Context, taskID uuid.UUID, status domain.PipelineTaskStatus, errMsg string, durationMs int) error {
	const q = `
		UPDATE pipeline_tasks
		SET    status       = $2,
		       error        = CASE WHEN $3 != '' THEN $3 ELSE error END,
		       duration_ms  = CASE WHEN $4 > 0 THEN $4 ELSE duration_ms END,
		       started_at   = CASE WHEN $2 = 'running' THEN NOW() ELSE started_at END,
		       completed_at = CASE WHEN $2 IN ('completed','failed') THEN NOW() ELSE completed_at END,
		       attempt      = CASE WHEN $2 = 'running' THEN attempt + 1 ELSE attempt END
		WHERE  id = $1`

	_, err := s.pool.Exec(ctx, q, taskID, status, errMsg, durationMs)
	if err != nil {
		return fmt.Errorf("store.UpdatePipelineTaskStatus: %w", err)
	}
	return nil
}

// UpdatePipelineRunStatus advances a pipeline run's overall status.
func (s *Store) UpdatePipelineRunStatus(ctx context.Context, runID uuid.UUID, status domain.PipelineRunStatus) (*domain.PipelineRun, error) {
	const q = `
		UPDATE pipeline_runs
		SET    status       = $2,
		       started_at   = CASE WHEN $2 = 'running' THEN NOW() ELSE started_at END,
		       completed_at = CASE WHEN $2 IN ('completed','failed') THEN NOW() ELSE completed_at END
		WHERE  id = $1
		RETURNING id, project_id, type, status, metadata, started_at, completed_at, error, created_at`

	var run domain.PipelineRun
	err := s.pool.QueryRow(ctx, q, runID, status).Scan(
		&run.ID, &run.ProjectID, &run.Type, &run.Status,
		&run.Metadata, &run.StartedAt, &run.CompletedAt, &run.Error, &run.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("store.UpdatePipelineRunStatus: %w", err)
	}
	return &run, nil
}

// GetPipelineRun returns a single pipeline run with its associated tasks.
func (s *Store) GetPipelineRun(ctx context.Context, runID uuid.UUID) (*domain.PipelineRun, error) {
	const runQ = `
		SELECT id, project_id, type, status, metadata, started_at, completed_at, error, created_at
		FROM   pipeline_runs
		WHERE  id = $1`

	var run domain.PipelineRun
	err := s.pool.QueryRow(ctx, runQ, runID).Scan(
		&run.ID, &run.ProjectID, &run.Type, &run.Status,
		&run.Metadata, &run.StartedAt, &run.CompletedAt, &run.Error, &run.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("store.GetPipelineRun: %w", err)
	}

	const taskQ = `
		SELECT id, pipeline_run_id, name, status, attempt, started_at, completed_at, duration_ms, error, sort_order
		FROM   pipeline_tasks
		WHERE  pipeline_run_id = $1
		ORDER  BY sort_order`

	rows, err := s.pool.Query(ctx, taskQ, runID)
	if err != nil {
		return nil, fmt.Errorf("store.GetPipelineRun: tasks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var task domain.PipelineTask
		if err := rows.Scan(
			&task.ID, &task.PipelineRunID, &task.Name, &task.Status,
			&task.Attempt, &task.StartedAt, &task.CompletedAt, &task.DurationMs, &task.Error, &task.SortOrder,
		); err != nil {
			return nil, fmt.Errorf("store.GetPipelineRun: scan task: %w", err)
		}
		run.Tasks = append(run.Tasks, task)
	}
	return &run, rows.Err()
}

// GetPipelineRunScoped returns a pipeline run scoped to a project (for API handlers).
func (s *Store) GetPipelineRunScoped(ctx context.Context, projectID, runID uuid.UUID) (*domain.PipelineRun, error) {
	const runQ = `
		SELECT id, project_id, type, status, metadata, started_at, completed_at, error, created_at
		FROM   pipeline_runs
		WHERE  id = $1 AND project_id = $2`

	var run domain.PipelineRun
	err := s.pool.QueryRow(ctx, runQ, runID, projectID).Scan(
		&run.ID, &run.ProjectID, &run.Type, &run.Status,
		&run.Metadata, &run.StartedAt, &run.CompletedAt, &run.Error, &run.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("store.GetPipelineRunScoped: %w", err)
	}

	const taskQ = `
		SELECT id, pipeline_run_id, name, status, attempt, started_at, completed_at, duration_ms, error, sort_order
		FROM   pipeline_tasks
		WHERE  pipeline_run_id = $1
		ORDER  BY sort_order`

	rows, err := s.pool.Query(ctx, taskQ, runID)
	if err != nil {
		return nil, fmt.Errorf("store.GetPipelineRunScoped: tasks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var task domain.PipelineTask
		if err := rows.Scan(
			&task.ID, &task.PipelineRunID, &task.Name, &task.Status,
			&task.Attempt, &task.StartedAt, &task.CompletedAt, &task.DurationMs, &task.Error, &task.SortOrder,
		); err != nil {
			return nil, fmt.Errorf("store.GetPipelineRunScoped: scan task: %w", err)
		}
		run.Tasks = append(run.Tasks, task)
	}
	return &run, rows.Err()
}

// ListProjectPipelines returns a paginated list of pipeline runs for a project.
func (s *Store) ListProjectPipelines(ctx context.Context, projectID uuid.UUID, pp PageParams) ([]domain.PipelineRun, int, error) {
	const countQ = `SELECT COUNT(*) FROM pipeline_runs WHERE project_id = $1`
	var total int
	if err := s.pool.QueryRow(ctx, countQ, projectID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store.ListProjectPipelines count: %w", err)
	}

	const q = `
		SELECT id, project_id, type, status, metadata, started_at, completed_at, error, created_at
		FROM   pipeline_runs
		WHERE  project_id = $1
		ORDER  BY created_at DESC
		LIMIT  $2 OFFSET $3`

	rows, err := s.pool.Query(ctx, q, projectID, pp.Limit, pp.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("store.ListProjectPipelines: %w", err)
	}
	defer rows.Close()

	var runs []domain.PipelineRun
	for rows.Next() {
		var run domain.PipelineRun
		if err := rows.Scan(
			&run.ID, &run.ProjectID, &run.Type, &run.Status,
			&run.Metadata, &run.StartedAt, &run.CompletedAt, &run.Error, &run.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("store.ListProjectPipelines: scan: %w", err)
		}
		runs = append(runs, run)
	}
	return runs, total, rows.Err()
}

// GetQueueDepths returns a formatted string summarising River queue depths
// from the river_jobs table. Used by the operator health endpoint.
func (s *Store) GetQueueDepths(ctx context.Context) (string, error) {
	const q = `
		SELECT state, COUNT(*) AS cnt
		FROM   river_jobs
		GROUP  BY state
		ORDER  BY state`

	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		// river_jobs may not exist in all environments; treat as non-fatal.
		return "unavailable", nil
	}
	defer rows.Close()

	type row struct {
		State string
		Count int64
	}

	var parts []string
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.State, &r.Count); err != nil {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%d", r.State, r.Count))
	}

	if len(parts) == 0 {
		return "empty", nil
	}
	return strings.Join(parts, ", "), nil
}
