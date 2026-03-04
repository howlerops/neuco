package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/neuco-ai/neuco/internal/domain"
)

const specColumns = `
	id, candidate_id, project_id, version,
	problem_statement, proposed_solution,
	user_stories, acceptance_criteria, out_of_scope,
	ui_changes, data_model_changes, open_questions,
	generated_by, created_at`

// CreateSpec inserts a new spec version. The version number must be supplied
// by the caller; use GetSpecByCandidate to find the current max version first.
func (s *Store) CreateSpec(ctx context.Context, spec domain.Spec) (domain.Spec, error) {
	userStoriesJSON, err := json.Marshal(spec.UserStories)
	if err != nil {
		return domain.Spec{}, fmt.Errorf("store.CreateSpec: marshal user_stories: %w", err)
	}
	criteriaJSON, err := json.Marshal(spec.AcceptanceCriteria)
	if err != nil {
		return domain.Spec{}, fmt.Errorf("store.CreateSpec: marshal acceptance_criteria: %w", err)
	}
	outOfScopeJSON, err := json.Marshal(spec.OutOfScope)
	if err != nil {
		return domain.Spec{}, fmt.Errorf("store.CreateSpec: marshal out_of_scope: %w", err)
	}
	openQJSON, err := json.Marshal(spec.OpenQuestions)
	if err != nil {
		return domain.Spec{}, fmt.Errorf("store.CreateSpec: marshal open_questions: %w", err)
	}

	const q = `
		INSERT INTO specs
		       (candidate_id, project_id, version,
		        problem_statement, proposed_solution,
		        user_stories, acceptance_criteria, out_of_scope,
		        ui_changes, data_model_changes, open_questions,
		        generated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING ` + specColumns

	row := s.pool.QueryRow(ctx, q,
		spec.CandidateID,
		spec.ProjectID,
		spec.Version,
		spec.ProblemStatement,
		spec.ProposedSolution,
		userStoriesJSON,
		criteriaJSON,
		outOfScopeJSON,
		spec.UIChanges,
		spec.DataModelChanges,
		openQJSON,
		spec.GeneratedBy,
	)
	out, err := scanSpec(row)
	if err != nil {
		return domain.Spec{}, fmt.Errorf("store.CreateSpec: %w", err)
	}
	return out, nil
}

// GetSpec returns a specific version of a spec scoped to projectID.
func (s *Store) GetSpec(ctx context.Context, projectID, specID uuid.UUID) (domain.Spec, error) {
	const q = `
		SELECT ` + specColumns + `
		FROM   specs
		WHERE  id = $1 AND project_id = $2`

	row := s.pool.QueryRow(ctx, q, specID, projectID)
	spec, err := scanSpec(row)
	if err != nil {
		return domain.Spec{}, fmt.Errorf("store.GetSpec: %w", err)
	}
	return spec, nil
}

// GetSpecByCandidate returns the latest version of the spec for a candidate.
func (s *Store) GetSpecByCandidate(ctx context.Context, projectID, candidateID uuid.UUID) (domain.Spec, error) {
	const q = `
		SELECT ` + specColumns + `
		FROM   specs
		WHERE  candidate_id = $1 AND project_id = $2
		ORDER  BY version DESC
		LIMIT  1`

	row := s.pool.QueryRow(ctx, q, candidateID, projectID)
	spec, err := scanSpec(row)
	if err != nil {
		return domain.Spec{}, fmt.Errorf("store.GetSpecByCandidate: %w", err)
	}
	return spec, nil
}

// UpdateSpec creates a new version of an existing spec. The version field is
// automatically incremented by reading the current max version inside a
// transaction, ensuring there are no gaps or conflicts.
func (s *Store) UpdateSpec(ctx context.Context, projectID, candidateID uuid.UUID, patch domain.Spec) (domain.Spec, error) {
	var result domain.Spec
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		// Lock the candidate row to serialise concurrent spec updates.
		const lockQ = `
			SELECT version FROM specs
			WHERE  candidate_id = $1 AND project_id = $2
			ORDER  BY version DESC
			LIMIT  1
			FOR UPDATE`
		var currentVersion int
		if err := tx.QueryRow(ctx, lockQ, candidateID, projectID).Scan(&currentVersion); err != nil {
			return fmt.Errorf("get current version: %w", err)
		}
		patch.Version = currentVersion + 1
		patch.CandidateID = candidateID
		patch.ProjectID = projectID

		userStoriesJSON, err := json.Marshal(patch.UserStories)
		if err != nil {
			return fmt.Errorf("marshal user_stories: %w", err)
		}
		criteriaJSON, err := json.Marshal(patch.AcceptanceCriteria)
		if err != nil {
			return fmt.Errorf("marshal acceptance_criteria: %w", err)
		}
		outOfScopeJSON, err := json.Marshal(patch.OutOfScope)
		if err != nil {
			return fmt.Errorf("marshal out_of_scope: %w", err)
		}
		openQJSON, err := json.Marshal(patch.OpenQuestions)
		if err != nil {
			return fmt.Errorf("marshal open_questions: %w", err)
		}

		const insertQ = `
			INSERT INTO specs
			       (candidate_id, project_id, version,
			        problem_statement, proposed_solution,
			        user_stories, acceptance_criteria, out_of_scope,
			        ui_changes, data_model_changes, open_questions,
			        generated_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			RETURNING ` + specColumns

		row := tx.QueryRow(ctx, insertQ,
			patch.CandidateID,
			patch.ProjectID,
			patch.Version,
			patch.ProblemStatement,
			patch.ProposedSolution,
			userStoriesJSON,
			criteriaJSON,
			outOfScopeJSON,
			patch.UIChanges,
			patch.DataModelChanges,
			openQJSON,
			patch.GeneratedBy,
		)
		out, err := scanSpec(row)
		if err != nil {
			return fmt.Errorf("insert: %w", err)
		}
		result = out
		return nil
	})
	if err != nil {
		return domain.Spec{}, fmt.Errorf("store.UpdateSpec: %w", err)
	}
	return result, nil
}

// ListSpecVersions returns all versions of the spec for a candidate, oldest
// first, so callers can show a version history.
func (s *Store) ListSpecVersions(ctx context.Context, projectID, candidateID uuid.UUID) ([]domain.Spec, error) {
	const q = `
		SELECT ` + specColumns + `
		FROM   specs
		WHERE  candidate_id = $1 AND project_id = $2
		ORDER  BY version ASC`

	rows, err := s.pool.Query(ctx, q, candidateID, projectID)
	if err != nil {
		return nil, fmt.Errorf("store.ListSpecVersions: %w", err)
	}
	defer rows.Close()

	var specs []domain.Spec
	for rows.Next() {
		spec, err := scanSpec(rows)
		if err != nil {
			return nil, fmt.Errorf("store.ListSpecVersions: scan: %w", err)
		}
		specs = append(specs, spec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.ListSpecVersions: rows: %w", err)
	}
	return specs, nil
}

// scanSpec reads a single Spec from any pgx row-like value.
func scanSpec(row pgx.Row) (domain.Spec, error) {
	var spec domain.Spec
	var userStoriesRaw, criteriaRaw, outOfScopeRaw, openQRaw []byte
	err := row.Scan(
		&spec.ID,
		&spec.CandidateID,
		&spec.ProjectID,
		&spec.Version,
		&spec.ProblemStatement,
		&spec.ProposedSolution,
		&userStoriesRaw,
		&criteriaRaw,
		&outOfScopeRaw,
		&spec.UIChanges,
		&spec.DataModelChanges,
		&openQRaw,
		&spec.GeneratedBy,
		&spec.CreatedAt,
	)
	if err != nil {
		return domain.Spec{}, err
	}

	if err := json.Unmarshal(userStoriesRaw, &spec.UserStories); err != nil {
		return domain.Spec{}, fmt.Errorf("unmarshal user_stories: %w", err)
	}
	if err := json.Unmarshal(criteriaRaw, &spec.AcceptanceCriteria); err != nil {
		return domain.Spec{}, fmt.Errorf("unmarshal acceptance_criteria: %w", err)
	}
	if err := json.Unmarshal(outOfScopeRaw, &spec.OutOfScope); err != nil {
		return domain.Spec{}, fmt.Errorf("unmarshal out_of_scope: %w", err)
	}
	if err := json.Unmarshal(openQRaw, &spec.OpenQuestions); err != nil {
		return domain.Spec{}, fmt.Errorf("unmarshal open_questions: %w", err)
	}
	return spec, nil
}
