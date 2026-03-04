package domain

import (
	"time"

	"github.com/google/uuid"
)

// Integration represents a connected external service for a project.
// Matches DB schema: id, project_id, provider, webhook_secret, config, last_sync_at, is_active, created_at
type Integration struct {
	ID            uuid.UUID      `json:"id"`
	ProjectID     uuid.UUID      `json:"project_id"`
	Provider      string         `json:"provider"`
	WebhookSecret string         `json:"webhook_secret,omitempty"`
	Config        map[string]any `json:"config,omitempty"`
	LastSyncAt    *time.Time     `json:"last_sync_at,omitempty"`
	IsActive      bool           `json:"is_active"`
	CreatedAt     time.Time      `json:"created_at"`
}
