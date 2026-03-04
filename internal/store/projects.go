package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/neuco-ai/neuco/internal/domain"
)

// CreateProject inserts a new project into the database and returns it.
func (s *Store) CreateProject(
	ctx context.Context,
	orgID uuid.UUID,
	name string,
	githubRepo string,
	framework domain.ProjectFramework,
	styling domain.ProjectStyling,
	createdBy uuid.UUID,
) (domain.Project, error) {
	const q = `
		INSERT INTO projects (org_id, name, github_repo, framework, styling, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, org_id, name, github_repo, framework, styling, created_by, created_at`

	row := s.pool.QueryRow(ctx, q, orgID, name, githubRepo, framework, styling, createdBy)
	p, err := scanProject(row)
	if err != nil {
		return domain.Project{}, fmt.Errorf("store.CreateProject: %w", err)
	}
	return p, nil
}

// GetProject returns a single project. The orgID parameter is required to
// enforce tenant isolation — the query will return pgx.ErrNoRows if the
// project exists but belongs to a different org.
func (s *Store) GetProject(ctx context.Context, orgID, projectID uuid.UUID) (domain.Project, error) {
	const q = `
		SELECT id, org_id, name, github_repo, framework, styling, created_by, created_at
		FROM   projects
		WHERE  id = $1 AND org_id = $2`

	row := s.pool.QueryRow(ctx, q, projectID, orgID)
	p, err := scanProject(row)
	if err != nil {
		return domain.Project{}, fmt.Errorf("store.GetProject: %w", err)
	}
	return p, nil
}

// ListOrgProjects returns all projects that belong to the supplied org.
func (s *Store) ListOrgProjects(ctx context.Context, orgID uuid.UUID) ([]domain.Project, error) {
	const q = `
		SELECT id, org_id, name, github_repo, framework, styling, created_by, created_at
		FROM   projects
		WHERE  org_id = $1
		ORDER  BY name`

	rows, err := s.pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, fmt.Errorf("store.ListOrgProjects: %w", err)
	}
	defer rows.Close()

	var projects []domain.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, fmt.Errorf("store.ListOrgProjects: scan: %w", err)
		}
		projects = append(projects, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.ListOrgProjects: rows: %w", err)
	}
	return projects, nil
}

// UpdateProject modifies mutable project fields. Only non-nil fields are
// updated, keeping the PATCH semantics clean.
func (s *Store) UpdateProject(
	ctx context.Context,
	orgID, projectID uuid.UUID,
	name *string,
	githubRepo *string,
	framework *domain.ProjectFramework,
	styling *domain.ProjectStyling,
) (domain.Project, error) {
	const q = `
		UPDATE projects
		SET    name        = COALESCE($3, name),
		       github_repo = COALESCE($4, github_repo),
		       framework   = COALESCE($5, framework),
		       styling     = COALESCE($6, styling)
		WHERE  id = $1 AND org_id = $2
		RETURNING id, org_id, name, github_repo, framework, styling, created_by, created_at`

	row := s.pool.QueryRow(ctx, q, projectID, orgID, name, githubRepo, framework, styling)
	p, err := scanProject(row)
	if err != nil {
		return domain.Project{}, fmt.Errorf("store.UpdateProject: %w", err)
	}
	return p, nil
}

// DeleteProject permanently removes a project and all of its associated data
// via ON DELETE CASCADE foreign key constraints.
func (s *Store) DeleteProject(ctx context.Context, orgID, projectID uuid.UUID) error {
	const q = `DELETE FROM projects WHERE id = $1 AND org_id = $2`
	ct, err := s.pool.Exec(ctx, q, projectID, orgID)
	if err != nil {
		return fmt.Errorf("store.DeleteProject: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("store.DeleteProject: project %s not found in org %s", projectID, orgID)
	}
	return nil
}

// scanProject reads a single Project from any pgx row-like value.
func scanProject(row pgx.Row) (domain.Project, error) {
	var p domain.Project
	err := row.Scan(
		&p.ID,
		&p.OrgID,
		&p.Name,
		&p.GitHubRepo,
		&p.Framework,
		&p.Styling,
		&p.CreatedBy,
		&p.CreatedAt,
	)
	if err != nil {
		return domain.Project{}, err
	}
	return p, nil
}
