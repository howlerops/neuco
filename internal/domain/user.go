package domain

import (
	"time"

	"github.com/google/uuid"
)

// User represents an authenticated Neuco user. Users are created (or updated)
// on each GitHub OAuth login via UpsertUser.
type User struct {
	ID          uuid.UUID `json:"id"`
	GitHubID    string    `json:"github_id"`
	GitHubLogin string    `json:"github_login"`
	Email       string    `json:"email"`
	AvatarURL   string    `json:"avatar_url"`
	CreatedAt   time.Time `json:"created_at"`
}
