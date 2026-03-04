package domain

import (
	"time"

	"github.com/google/uuid"
)

// UserStory represents a single "As a … I want … so that …" statement within a spec.
type UserStory struct {
	Role   string `json:"role"`
	Want   string `json:"want"`
	SoThat string `json:"so_that"`
}

// Spec is a structured product specification generated from a FeatureCandidate.
// Each edit creates a new version row; the latest version is the canonical spec.
type Spec struct {
	ID                  uuid.UUID   `json:"id"`
	CandidateID         uuid.UUID   `json:"candidate_id"`
	ProjectID           uuid.UUID   `json:"project_id"`
	Version             int         `json:"version"`
	ProblemStatement    string      `json:"problem_statement"`
	ProposedSolution    string      `json:"proposed_solution"`
	UserStories         []UserStory `json:"user_stories"`
	AcceptanceCriteria  []string    `json:"acceptance_criteria"`
	OutOfScope          []string    `json:"out_of_scope"`
	UIChanges           string      `json:"ui_changes"`
	DataModelChanges    string      `json:"data_model_changes"`
	OpenQuestions       []string    `json:"open_questions"`
	GeneratedBy         *uuid.UUID  `json:"generated_by,omitempty"` // user ID if manually edited; nil if AI-generated
	CreatedAt           time.Time   `json:"created_at"`
}
