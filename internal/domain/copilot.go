package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CopilotNoteTargetType identifies the entity the copilot note refers to.
type CopilotNoteTargetType string

const (
	CopilotNoteTargetCandidate CopilotNoteTargetType = "candidate"
	CopilotNoteTargetSignal    CopilotNoteTargetType = "signal_batch"
	CopilotNoteTargetSpec      CopilotNoteTargetType = "spec"
	CopilotNoteTargetGeneration CopilotNoteTargetType = "generation"
	CopilotNoteTargetSynthesis CopilotNoteTargetType = "synthesis"
)

// CopilotNoteType categorizes the kind of copilot note.
type CopilotNoteType string

const (
	CopilotNoteTypeReview     CopilotNoteType = "review"
	CopilotNoteTypeRisk       CopilotNoteType = "risk"
	CopilotNoteTypeSuggestion CopilotNoteType = "suggestion"
	CopilotNoteTypeInsight    CopilotNoteType = "insight"
)

// CopilotNote is an AI-generated insight or suggestion attached to a specific
// entity within a project. Notes can be dismissed by the user.
type CopilotNote struct {
	ID         uuid.UUID             `json:"id"`
	ProjectID  uuid.UUID             `json:"project_id"`
	TargetType CopilotNoteTargetType `json:"target_type"`
	TargetID   uuid.UUID             `json:"target_id"`
	NoteType   CopilotNoteType       `json:"note_type"`
	Content    string                `json:"content"`
	Metadata   json.RawMessage       `json:"metadata,omitempty"`
	Dismissed  bool                  `json:"dismissed"`
	CreatedAt  time.Time             `json:"created_at"`
}
