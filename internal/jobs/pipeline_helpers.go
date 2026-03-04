package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/neuco-ai/neuco/internal/domain"
	"github.com/neuco-ai/neuco/internal/store"
)

// StartTask marks a pipeline task as running.
func StartTask(ctx context.Context, s *store.Store, taskID uuid.UUID) {
	if taskID == uuid.Nil {
		return
	}
	if err := s.UpdatePipelineTaskStatus(ctx, taskID, domain.PipelineTaskStatusRunning, "", 0); err != nil {
		slog.Error("failed to mark task running", "task_id", taskID, "error", err)
	}
}

// CompleteTask marks a pipeline task as completed with its duration.
func CompleteTask(ctx context.Context, s *store.Store, taskID uuid.UUID, startTime time.Time) {
	if taskID == uuid.Nil {
		return
	}
	durationMs := int(time.Since(startTime).Milliseconds())
	if err := s.UpdatePipelineTaskStatus(ctx, taskID, domain.PipelineTaskStatusCompleted, "", durationMs); err != nil {
		slog.Error("failed to mark task completed", "task_id", taskID, "error", err)
	}
}

// FailTask marks a pipeline task as failed with the error message.
func FailTask(ctx context.Context, s *store.Store, taskID uuid.UUID, taskErr error) {
	if taskID == uuid.Nil {
		return
	}
	errMsg := ""
	if taskErr != nil {
		errMsg = taskErr.Error()
	}
	if err := s.UpdatePipelineTaskStatus(ctx, taskID, domain.PipelineTaskStatusFailed, errMsg, 0); err != nil {
		slog.Error("failed to mark task failed", "task_id", taskID, "error", err)
	}
}

// CheckPipelineCompletion checks if all tasks in a pipeline run are done
// and updates the run status accordingly.
func CheckPipelineCompletion(ctx context.Context, s *store.Store, runID uuid.UUID) {
	if runID == uuid.Nil {
		return
	}
	run, err := s.GetPipelineRun(ctx, runID)
	if err != nil {
		slog.Error("failed to get pipeline run for completion check", "run_id", runID, "error", err)
		return
	}

	allCompleted := true
	anyFailed := false
	for _, task := range run.Tasks {
		if task.Status == domain.PipelineTaskStatusFailed {
			anyFailed = true
			break
		}
		if task.Status != domain.PipelineTaskStatusCompleted {
			allCompleted = false
		}
	}

	if anyFailed {
		if _, err := s.UpdatePipelineRunStatus(ctx, runID, domain.PipelineRunStatusFailed); err != nil {
			slog.Error("failed to update pipeline run status", "run_id", runID, "error", err)
		}
		if err := s.UpdatePipelineRunError(ctx, runID, "one or more tasks failed"); err != nil {
			slog.Error("failed to update pipeline run error", "run_id", runID, "error", err)
		}
	} else if allCompleted {
		if _, err := s.UpdatePipelineRunStatus(ctx, runID, domain.PipelineRunStatusCompleted); err != nil {
			slog.Error("failed to update pipeline run status", "run_id", runID, "error", err)
		}
	}
}

// CreateSynthesisPipeline creates a pipeline run and tasks for a synthesis workflow.
func CreateSynthesisPipeline(ctx context.Context, s *store.Store, projectID uuid.UUID) (uuid.UUID, []uuid.UUID, error) {
	run, err := s.CreatePipelineRun(ctx, projectID, domain.PipelineTypeSynthesis, nil)
	if err != nil {
		return uuid.Nil, nil, err
	}

	taskNames := []string{"fetch_signals", "embed_missing", "cluster_themes", "name_themes", "score_candidates", "write_candidates"}
	taskIDs := make([]uuid.UUID, len(taskNames))
	for i, name := range taskNames {
		task, err := s.CreatePipelineTask(ctx, run.ID, name, i)
		if err != nil {
			return uuid.Nil, nil, err
		}
		taskIDs[i] = task.ID
	}

	return run.ID, taskIDs, nil
}

// CreateCodegenPipeline creates a pipeline run and tasks for a codegen workflow.
func CreateCodegenPipeline(ctx context.Context, s *store.Store, projectID uuid.UUID, specID uuid.UUID) (uuid.UUID, []uuid.UUID, error) {
	metadata := map[string]string{
		"spec_id": specID.String(),
	}
	run, err := s.CreatePipelineRun(ctx, projectID, domain.PipelineTypeCodegen, metadata)
	if err != nil {
		return uuid.Nil, nil, err
	}

	taskNames := []string{"fetch_spec", "index_repo", "build_context", "generate_code", "create_pr", "notify"}
	taskIDs := make([]uuid.UUID, len(taskNames))
	for i, name := range taskNames {
		task, err := s.CreatePipelineTask(ctx, run.ID, name, i)
		if err != nil {
			return uuid.Nil, nil, err
		}
		taskIDs[i] = task.ID
	}

	return run.ID, taskIDs, nil
}

// CreateSpecGenPipeline creates a pipeline run for spec generation.
func CreateSpecGenPipeline(ctx context.Context, s *store.Store, projectID uuid.UUID, candidateID uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	metadata := map[string]string{
		"candidate_id": candidateID.String(),
	}
	run, err := s.CreatePipelineRun(ctx, projectID, domain.PipelineTypeSpecGen, metadata)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	task, err := s.CreatePipelineTask(ctx, run.ID, "generate_spec", 0)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	return run.ID, task.ID, nil
}

// CreateIngestPipeline creates a pipeline run for signal ingestion.
func CreateIngestPipeline(ctx context.Context, s *store.Store, projectID uuid.UUID) (uuid.UUID, []uuid.UUID, error) {
	run, err := s.CreatePipelineRun(ctx, projectID, domain.PipelineTypeIngest, nil)
	if err != nil {
		return uuid.Nil, nil, err
	}

	taskNames := []string{"ingest", "embed"}
	taskIDs := make([]uuid.UUID, len(taskNames))
	for i, name := range taskNames {
		task, err := s.CreatePipelineTask(ctx, run.ID, name, i)
		if err != nil {
			return uuid.Nil, nil, err
		}
		taskIDs[i] = task.ID
	}

	return run.ID, taskIDs, nil
}

// CreateNangoSyncPipeline creates a pipeline run for a Nango integration sync.
// The run contains two tasks: "nango_sync" (fetch signals from the provider)
// and "embed" (generate vector embeddings for the fetched signals).
func CreateNangoSyncPipeline(ctx context.Context, s *store.Store, projectID uuid.UUID) (uuid.UUID, []uuid.UUID, error) {
	run, err := s.CreatePipelineRun(ctx, projectID, domain.PipelineTypeNangoSync, nil)
	if err != nil {
		return uuid.Nil, nil, err
	}

	taskNames := []string{"nango_sync", "embed"}
	taskIDs := make([]uuid.UUID, len(taskNames))
	for i, name := range taskNames {
		task, err := s.CreatePipelineTask(ctx, run.ID, name, i)
		if err != nil {
			return uuid.Nil, nil, err
		}
		taskIDs[i] = task.ID
	}

	return run.ID, taskIDs, nil
}
