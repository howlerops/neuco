package store

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/neuco-ai/neuco/internal/domain"
)

// UpsertUser inserts a new user or updates the login, email and avatar_url of
// the matching GitHub account on conflict. Returns the full User row.
// githubID is converted to string for the TEXT column in the database.
func (s *Store) UpsertUser(ctx context.Context, githubID int64, login, email, avatarURL string) (domain.User, error) {
	const q = `
		INSERT INTO users (github_id, github_login, email, avatar_url)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (github_id) DO UPDATE
			SET github_login = EXCLUDED.github_login,
			    email        = EXCLUDED.email,
			    avatar_url   = EXCLUDED.avatar_url
		RETURNING id, github_id, github_login, email, avatar_url, created_at`

	ghIDStr := strconv.FormatInt(githubID, 10)
	row := s.pool.QueryRow(ctx, q, ghIDStr, login, email, avatarURL)
	u, err := scanUser(row)
	if err != nil {
		return domain.User{}, fmt.Errorf("store.UpsertUser: %w", err)
	}
	return u, nil
}

// GetUserByID returns the user with the given UUID.
func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	const q = `
		SELECT id, github_id, github_login, email, avatar_url, created_at
		FROM   users
		WHERE  id = $1`

	row := s.pool.QueryRow(ctx, q, id)
	u, err := scanUser(row)
	if err != nil {
		return domain.User{}, fmt.Errorf("store.GetUserByID: %w", err)
	}
	return u, nil
}

// GetUserByGitHubID returns the user associated with the given GitHub account.
func (s *Store) GetUserByGitHubID(ctx context.Context, githubID int64) (domain.User, error) {
	const q = `
		SELECT id, github_id, github_login, email, avatar_url, created_at
		FROM   users
		WHERE  github_id = $1`

	ghIDStr := strconv.FormatInt(githubID, 10)
	row := s.pool.QueryRow(ctx, q, ghIDStr)
	u, err := scanUser(row)
	if err != nil {
		return domain.User{}, fmt.Errorf("store.GetUserByGitHubID: %w", err)
	}
	return u, nil
}

// SetUserGitHubToken persists the GitHub OAuth access token for a user.
// The token is stored in the github_token column of the users table.
func (s *Store) SetUserGitHubToken(ctx context.Context, userID uuid.UUID, token string) error {
	const q = `UPDATE users SET github_token = $1 WHERE id = $2`
	_, err := s.pool.Exec(ctx, q, token, userID)
	if err != nil {
		return fmt.Errorf("store.SetUserGitHubToken: %w", err)
	}
	return nil
}

// GetUserGitHubToken returns the stored GitHub OAuth access token for a user.
// Returns an empty string when no token has been stored yet (token column is NULL).
func (s *Store) GetUserGitHubToken(ctx context.Context, userID uuid.UUID) (string, error) {
	const q = `SELECT COALESCE(github_token, '') FROM users WHERE id = $1`
	var token string
	if err := s.pool.QueryRow(ctx, q, userID).Scan(&token); err != nil {
		return "", fmt.Errorf("store.GetUserGitHubToken: %w", err)
	}
	return token, nil
}

// scanUser reads a single User from any pgx row-like value.
func scanUser(row pgx.Row) (domain.User, error) {
	var u domain.User
	err := row.Scan(
		&u.ID,
		&u.GitHubID,
		&u.GitHubLogin,
		&u.Email,
		&u.AvatarURL,
		&u.CreatedAt,
	)
	if err != nil {
		return domain.User{}, err
	}
	return u, nil
}

// ListAllUsers returns every user in the system.
// This is an operator-only method.
func (s *Store) ListAllUsers(ctx context.Context) ([]domain.User, error) {
	const q = `
		SELECT id, github_id, github_login, email, avatar_url, created_at
		FROM   users
		ORDER  BY created_at DESC`

	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("store.ListAllUsers: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("store.ListAllUsers: scan: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.ListAllUsers: rows: %w", err)
	}
	return users, nil
}
