package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// SetOrgGitHubInstallation stores the GitHub App installation_id for an org.
// This is called when a user completes the GitHub App installation flow and
// GitHub redirects back with the installation_id query parameter.
func (s *Store) SetOrgGitHubInstallation(ctx context.Context, orgID uuid.UUID, installationID int64) error {
	const q = `
		UPDATE organizations
		SET    github_installation_id = $2,
		       updated_at             = NOW()
		WHERE  id = $1`

	ct, err := s.pool.Exec(ctx, q, orgID, installationID)
	if err != nil {
		return fmt.Errorf("store.SetOrgGitHubInstallation: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("store.SetOrgGitHubInstallation: org %s not found", orgID)
	}
	return nil
}

// GetOrgGitHubInstallation returns the GitHub App installation_id for an org.
// Returns pgx.ErrNoRows (wrapped) when the column is NULL (App not installed).
func (s *Store) GetOrgGitHubInstallation(ctx context.Context, orgID uuid.UUID) (int64, error) {
	const q = `SELECT github_installation_id FROM organizations WHERE id = $1`

	var id *int64
	if err := s.pool.QueryRow(ctx, q, orgID).Scan(&id); err != nil {
		return 0, fmt.Errorf("store.GetOrgGitHubInstallation: %w", err)
	}
	if id == nil {
		return 0, fmt.Errorf("store.GetOrgGitHubInstallation: %w: installation not set for org %s", pgx.ErrNoRows, orgID)
	}
	return *id, nil
}

// GetProjectGitHubInstallation looks up the org that owns projectID and returns
// its GitHub App installation_id. This is the main entry point used by workers
// that have a project ID but need an installation token.
func (s *Store) GetProjectGitHubInstallation(ctx context.Context, projectID uuid.UUID) (int64, error) {
	const q = `
		SELECT o.github_installation_id
		FROM   projects p
		JOIN   organizations o ON o.id = p.org_id
		WHERE  p.id = $1`

	var id *int64
	if err := s.pool.QueryRow(ctx, q, projectID).Scan(&id); err != nil {
		return 0, fmt.Errorf("store.GetProjectGitHubInstallation: %w", err)
	}
	if id == nil {
		return 0, fmt.Errorf("store.GetProjectGitHubInstallation: no GitHub App installation for project %s", projectID)
	}
	return *id, nil
}
