package domain

import (
	"time"

	"github.com/google/uuid"
)

// User represents an authenticated Neuco user. Users are created (or updated)
// on each GitHub or Google OAuth login via UpsertUser / UpsertUserByGoogle.
type User struct {
	ID          uuid.UUID `json:"id"`
	GitHubID    string    `json:"github_id"`
	GitHubLogin string    `json:"github_login"`
	GoogleID    string    `json:"google_id,omitempty"`
	Name        string    `json:"name,omitempty"`
	Email       string    `json:"email"`
	AvatarURL   string    `json:"avatar_url"`
	CreatedAt   time.Time `json:"created_at"`
}
