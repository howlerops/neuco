package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/neuco-ai/neuco/internal/domain"
)

const candidateColumns = `
	id, project_id, title, problem_summary, signal_count, score, status, suggested_at`

// UpsertCandidate inserts a new FeatureCandidate or updates it if a row with
// the same (project_id, cluster_id) already exists. This is the primary write
// path used by the synthesis pipeline.
func (s *Store) UpsertCandidate(ctx context.Context, c domain.FeatureCandidate) (domain.FeatureCandidate, error) {
	const q = `
		INSERT INTO feature_candidates
		       (id, project_id, title, problem_summary, signal_count, score, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE
			SET title            = EXCLUDED.title,
			    problem_summary  = EXCLUDED.problem_summary,
			    score            = EXCLUDED.score,
			    signal_count     = EXCLUDED.signal_count
		RETURNING ` + candidateColumns

	row := s.pool.QueryRow(ctx, q,
		c.ID,
		c.ProjectID,
		c.Title,
		c.ProblemSummary,
		c.SignalCount,
		c.Score,
		c.Status,
	)
	out, err := scanCandidate(row)
	if err != nil {
		return domain.FeatureCandidate{}, fmt.Errorf("store.UpsertCandidate: %w", err)
	}
	return out, nil
}

// GetCandidate returns a single FeatureCandidate scoped to projectID.
func (s *Store) GetCandidate(ctx context.Context, projectID, candidateID uuid.UUID) (domain.FeatureCandidate, error) {
	const q = `
		SELECT ` + candidateColumns + `
		FROM   feature_candidates
		WHERE  id = $1 AND project_id = $2`

	row := s.pool.QueryRow(ctx, q, candidateID, projectID)
	c, err := scanCandidate(row)
	if err != nil {
		return domain.FeatureCandidate{}, fmt.Errorf("store.GetCandidate: %w", err)
	}
	return c, nil
}

// ListProjectCandidates returns all candidates for a project sorted by score
// descending (highest-priority first).
func (s *Store) ListProjectCandidates(ctx context.Context, projectID uuid.UUID, pp PageParams) ([]domain.FeatureCandidate, int, error) {
	const countQ = `SELECT COUNT(*) FROM feature_candidates WHERE project_id = $1`
	var total int
	if err := s.pool.QueryRow(ctx, countQ, projectID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store.ListProjectCandidates count: %w", err)
	}

	const q = `
		SELECT ` + candidateColumns + `
		FROM   feature_candidates
		WHERE  project_id = $1
		ORDER  BY score DESC, suggested_at DESC
		LIMIT  $2 OFFSET $3`

	rows, err := s.pool.Query(ctx, q, projectID, pp.Limit, pp.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("store.ListProjectCandidates: %w", err)
	}
	defer rows.Close()

	var candidates []domain.FeatureCandidate
	for rows.Next() {
		c, err := scanCandidate(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("store.ListProjectCandidates: scan: %w", err)
		}
		candidates = append(candidates, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("store.ListProjectCandidates: rows: %w", err)
	}
	return candidates, total, nil
}

// UpdateCandidateStatus changes the workflow status of a candidate.
func (s *Store) UpdateCandidateStatus(ctx context.Context, projectID, candidateID uuid.UUID, status domain.CandidateStatus) (domain.FeatureCandidate, error) {
	const q = `
		UPDATE feature_candidates
		SET    status = $3
		WHERE  id = $1 AND project_id = $2
		RETURNING ` + candidateColumns

	row := s.pool.QueryRow(ctx, q, candidateID, projectID, status)
	c, err := scanCandidate(row)
	if err != nil {
		return domain.FeatureCandidate{}, fmt.Errorf("store.UpdateCandidateStatus: %w", err)
	}
	return c, nil
}

// scanCandidate reads a single FeatureCandidate from any pgx row-like value.
func scanCandidate(row pgx.Row) (domain.FeatureCandidate, error) {
	var c domain.FeatureCandidate
	err := row.Scan(
		&c.ID,
		&c.ProjectID,
		&c.Title,
		&c.ProblemSummary,
		&c.SignalCount,
		&c.Score,
		&c.Status,
		&c.CreatedAt,
	)
	if err != nil {
		return domain.FeatureCandidate{}, err
	}
	return c, nil
}
