package domain

import (
	"time"

	"github.com/google/uuid"
)

// GenerationStatus tracks the state of a code generation job.
type GenerationStatus string

const (
	GenerationStatusPending   GenerationStatus = "pending"
	GenerationStatusRunning   GenerationStatus = "running"
	GenerationStatusCompleted GenerationStatus = "completed"
	GenerationStatusFailed    GenerationStatus = "failed"
)

// GeneratedFile represents a single file produced by a generation run.
type GeneratedFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// Generation represents a single code-generation run tied to a Spec.
// It produces one or more files that implement the feature described in the Spec.
type Generation struct {
	ID            uuid.UUID        `json:"id"`
	ProjectID     uuid.UUID        `json:"project_id"`
	SpecID        uuid.UUID        `json:"spec_id"`
	PipelineRunID uuid.UUID        `json:"pipeline_run_id"`
	Status        GenerationStatus `json:"status"`
	BranchName    string           `json:"branch_name,omitempty"`
	PRNumber      *int             `json:"pr_number,omitempty"`
	PRURL         string           `json:"pr_url,omitempty"`
	Files         []GeneratedFile  `json:"files,omitempty"`
	ErrorMsg      string           `json:"error,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
	CompletedAt   *time.Time       `json:"completed_at,omitempty"`
}
