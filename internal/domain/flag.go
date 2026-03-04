package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// FeatureFlag represents a feature flag stored in the database.
// Flags gate rollout of new capabilities and are toggled by operators.
type FeatureFlag struct {
	Key         string          `json:"key"`
	Enabled     bool            `json:"enabled"`
	Description string          `json:"description"`
	Metadata    json.RawMessage `json:"metadata"`
	UpdatedAt   time.Time       `json:"updated_at"`
	UpdatedBy   *uuid.UUID      `json:"updated_by,omitempty"`
}
