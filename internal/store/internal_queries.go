package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/neuco-ai/neuco/internal/domain"
)

// Internal query methods used by workers. These skip tenant scoping because
// workers operate on job arguments that already include verified project IDs.
// These should NEVER be exposed through HTTP handlers.

// GetSpecInternal fetches a spec by ID without project scoping (for workers).
func (s *Store) GetSpecInternal(ctx context.Context, specID uuid.UUID) (*domain.Spec, error) {
	const q = `
		SELECT id, candidate_id, project_id, version,
		       problem_statement, proposed_solution,
		       user_stories, acceptance_criteria, out_of_scope,
		       ui_changes, data_model_changes, open_questions,
		       created_at
		FROM   specs
		WHERE  id = $1
		ORDER  BY version DESC
		LIMIT  1`

	var spec domain.Spec
	var userStoriesJSON, criteriaJSON, oosJSON, oqJSON []byte
	err := s.pool.QueryRow(ctx, q, specID).Scan(
		&spec.ID,
		&spec.CandidateID,
		&spec.ProjectID,
		&spec.Version,
		&spec.ProblemStatement,
		&spec.ProposedSolution,
		&userStoriesJSON,
		&criteriaJSON,
		&oosJSON,
		&spec.UIChanges,
		&spec.DataModelChanges,
		&oqJSON,
		&spec.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("store.GetSpecInternal: %w", err)
	}
	json.Unmarshal(userStoriesJSON, &spec.UserStories)
	json.Unmarshal(criteriaJSON, &spec.AcceptanceCriteria)
	json.Unmarshal(oosJSON, &spec.OutOfScope)
	json.Unmarshal(oqJSON, &spec.OpenQuestions)
	return &spec, nil
}

// GetCandidateInternal fetches a candidate by ID without project scoping.
func (s *Store) GetCandidateInternal(ctx context.Context, candidateID uuid.UUID) (*domain.FeatureCandidate, error) {
	const q = `
		SELECT id, project_id, title, problem_summary, signal_count, score, status, created_at
		FROM   feature_candidates
		WHERE  id = $1`

	var c domain.FeatureCandidate
	err := s.pool.QueryRow(ctx, q, candidateID).Scan(
		&c.ID, &c.ProjectID, &c.Title, &c.ProblemSummary,
		&c.SignalCount, &c.Score, &c.Status, &c.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("store.GetCandidateInternal: %w", err)
	}
	return &c, nil
}

// GetProjectInternal fetches a project by ID without org scoping.
func (s *Store) GetProjectInternal(ctx context.Context, projectID uuid.UUID) (*domain.Project, error) {
	const q = `
		SELECT id, org_id, name, github_repo, framework, styling, created_by, created_at
		FROM   projects
		WHERE  id = $1`

	var p domain.Project
	err := s.pool.QueryRow(ctx, q, projectID).Scan(
		&p.ID, &p.OrgID, &p.Name, &p.GitHubRepo,
		&p.Framework, &p.Styling, &p.CreatedBy, &p.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("store.GetProjectInternal: %w", err)
	}
	return &p, nil
}

// GetSignalInternal fetches a signal by ID without project scoping.
func (s *Store) GetSignalInternal(ctx context.Context, signalID uuid.UUID) (*domain.Signal, error) {
	const q = `
		SELECT id, project_id, source, source_ref, type, content, metadata, occurred_at, ingested_at
		FROM   signals
		WHERE  id = $1`

	var sig domain.Signal
	err := s.pool.QueryRow(ctx, q, signalID).Scan(
		&sig.ID, &sig.ProjectID, &sig.Source, &sig.SourceRef,
		&sig.Type, &sig.Content, &sig.Metadata,
		&sig.OccurredAt, &sig.IngestedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("store.GetSignalInternal: %w", err)
	}
	return &sig, nil
}

// GetCandidateSignals fetches signals linked to a candidate, ordered by relevance.
func (s *Store) GetCandidateSignals(ctx context.Context, candidateID uuid.UUID, limit int) ([]domain.Signal, error) {
	const q = `
		SELECT s.id, s.project_id, s.source, s.source_ref, s.type, s.content, s.metadata,
		       s.occurred_at, s.ingested_at
		FROM   signals s
		JOIN   candidate_signals cs ON cs.signal_id = s.id
		WHERE  cs.candidate_id = $1
		ORDER  BY cs.relevance DESC
		LIMIT  $2`

	rows, err := s.pool.Query(ctx, q, candidateID, limit)
	if err != nil {
		return nil, fmt.Errorf("store.GetCandidateSignals: %w", err)
	}
	defer rows.Close()

	var signals []domain.Signal
	for rows.Next() {
		var sig domain.Signal
		if err := rows.Scan(
			&sig.ID, &sig.ProjectID, &sig.Source, &sig.SourceRef,
			&sig.Type, &sig.Content, &sig.Metadata,
			&sig.OccurredAt, &sig.IngestedAt,
		); err != nil {
			return nil, fmt.Errorf("store.GetCandidateSignals: scan: %w", err)
		}
		signals = append(signals, sig)
	}
	return signals, rows.Err()
}

// LinkCandidateSignal creates a link between a candidate and a signal.
func (s *Store) LinkCandidateSignal(ctx context.Context, candidateID, signalID uuid.UUID, relevance float64) error {
	const q = `
		INSERT INTO candidate_signals (candidate_id, signal_id, relevance)
		VALUES ($1, $2, $3)
		ON CONFLICT (candidate_id, signal_id) DO UPDATE SET relevance = $3`

	_, err := s.pool.Exec(ctx, q, candidateID, signalID, relevance)
	if err != nil {
		return fmt.Errorf("store.LinkCandidateSignal: %w", err)
	}
	return nil
}

// UpdateCandidateTheme updates the title and problem summary of a candidate.
func (s *Store) UpdateCandidateTheme(ctx context.Context, candidateID uuid.UUID, title, summary string) error {
	const q = `
		UPDATE feature_candidates
		SET    title = $2, problem_summary = $3
		WHERE  id = $1`

	_, err := s.pool.Exec(ctx, q, candidateID, title, summary)
	if err != nil {
		return fmt.Errorf("store.UpdateCandidateTheme: %w", err)
	}
	return nil
}

// UpdateCandidateScore updates the score of a candidate.
func (s *Store) UpdateCandidateScore(ctx context.Context, candidateID uuid.UUID, score float64) error {
	const q = `UPDATE feature_candidates SET score = $2 WHERE id = $1`
	_, err := s.pool.Exec(ctx, q, candidateID, score)
	if err != nil {
		return fmt.Errorf("store.UpdateCandidateScore: %w", err)
	}
	return nil
}

// ListEmbeddedSignals returns signals that have embeddings for a project.
func (s *Store) ListEmbeddedSignals(ctx context.Context, projectID uuid.UUID, limit int) ([]domain.Signal, error) {
	const q = `
		SELECT id, project_id, source, source_ref, type, content, metadata,
		       occurred_at, ingested_at, embedding
		FROM   signals
		WHERE  project_id = $1 AND embedding IS NOT NULL
		ORDER  BY ingested_at DESC
		LIMIT  $2`

	rows, err := s.pool.Query(ctx, q, projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("store.ListEmbeddedSignals: %w", err)
	}
	defer rows.Close()

	var signals []domain.Signal
	for rows.Next() {
		var sig domain.Signal
		if err := rows.Scan(
			&sig.ID, &sig.ProjectID, &sig.Source, &sig.SourceRef,
			&sig.Type, &sig.Content, &sig.Metadata,
			&sig.OccurredAt, &sig.IngestedAt, &sig.Embedding,
		); err != nil {
			return nil, fmt.Errorf("store.ListEmbeddedSignals: scan: %w", err)
		}
		signals = append(signals, sig)
	}
	return signals, rows.Err()
}

// ListAllActiveProjects returns all projects (used by weekly digest cron).
func (s *Store) ListAllActiveProjects(ctx context.Context) ([]domain.Project, error) {
	const q = `
		SELECT id, org_id, name, github_repo, framework, styling, created_by, created_at
		FROM   projects
		ORDER  BY created_at`

	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("store.ListAllActiveProjects: %w", err)
	}
	defer rows.Close()

	var projects []domain.Project
	for rows.Next() {
		var p domain.Project
		if err := rows.Scan(
			&p.ID, &p.OrgID, &p.Name, &p.GitHubRepo,
			&p.Framework, &p.Styling, &p.CreatedBy, &p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("store.ListAllActiveProjects: scan: %w", err)
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// GetProjectStats returns aggregate counters for the project dashboard.
func (s *Store) GetProjectStats(ctx context.Context, projectID uuid.UUID) (ProjectStats, error) {
	const q = `
		SELECT
			(SELECT COUNT(*) FROM signals           WHERE project_id = $1)::int AS signal_count,
			(SELECT COUNT(*) FROM feature_candidates WHERE project_id = $1)::int AS candidate_count,
			(SELECT COUNT(*) FROM generations        WHERE project_id = $1)::int AS generation_count,
			(SELECT COUNT(*) FROM pipeline_runs      WHERE project_id = $1)::int AS pipeline_count`

	var stats ProjectStats
	err := s.pool.QueryRow(ctx, q, projectID).Scan(
		&stats.SignalCount,
		&stats.CandidateCount,
		&stats.GenerationCount,
		&stats.PipelineCount,
	)
	if err != nil {
		return ProjectStats{}, fmt.Errorf("store.GetProjectStats: %w", err)
	}
	return stats, nil
}

// UpdatePipelineRunError updates the error field of a pipeline run.
func (s *Store) UpdatePipelineRunError(ctx context.Context, runID uuid.UUID, errMsg string) error {
	const q = `UPDATE pipeline_runs SET error = $2 WHERE id = $1`
	_, err := s.pool.Exec(ctx, q, runID, errMsg)
	return err
}

// UpdatePipelineRunMetadata replaces the metadata JSONB column of a pipeline
// run. Workers use this to pass structured data (e.g. a repo index) to
// downstream workers without an additional database table.
// metadata must be a valid JSON object (json.RawMessage).
func (s *Store) UpdatePipelineRunMetadata(ctx context.Context, runID uuid.UUID, metadata json.RawMessage) error {
	const q = `UPDATE pipeline_runs SET metadata = $2 WHERE id = $1`
	_, err := s.pool.Exec(ctx, q, runID, metadata)
	if err != nil {
		return fmt.Errorf("store.UpdatePipelineRunMetadata: %w", err)
	}
	return nil
}
