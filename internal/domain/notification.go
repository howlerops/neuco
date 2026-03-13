package domain

import (
	"time"

	"github.com/google/uuid"
)

// NotificationType identifies the kind of in-app notification.
type NotificationType string

const (
	NotificationTypePipelineCompleted NotificationType = "pipeline_completed"
	NotificationTypePipelineFailed    NotificationType = "pipeline_failed"
	NotificationTypeNewCandidate      NotificationType = "new_candidate"
	NotificationTypeCopilotInsight    NotificationType = "copilot_insight"
	NotificationTypeNewSignalBatch    NotificationType = "new_signal_batch"
	NotificationTypePRCreated         NotificationType = "pr_created"
)

// Notification is an in-app notification for a user within an organisation.
type Notification struct {
	ID        uuid.UUID        `json:"id"`
	OrgID     uuid.UUID        `json:"org_id"`
	UserID    *uuid.UUID       `json:"user_id,omitempty"`
	Type      NotificationType `json:"type"`
	Title     string           `json:"title"`
	Body      string           `json:"body"`
	Link      string           `json:"link"`
	ReadAt    *time.Time       `json:"read_at,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
}
