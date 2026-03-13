package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ContextCategory classifies the kind of accumulated project insight.
type ContextCategory string

const (
	ContextCategoryInsight     ContextCategory = "insight"
	ContextCategoryTheme       ContextCategory = "theme"
	ContextCategoryDecision    ContextCategory = "decision"
	ContextCategoryRisk        ContextCategory = "risk"
	ContextCategoryOpportunity ContextCategory = "opportunity"
)

// ProjectContext is a single accumulated insight that persists across synthesis
// runs. Over time these form the project's institutional memory, enabling the
// AI to build on prior analysis rather than starting from scratch.
type ProjectContext struct {
	ID          uuid.UUID       `json:"id"`
	ProjectID   uuid.UUID       `json:"project_id"`
	Category    ContextCategory `json:"category"`
	Title       string          `json:"title"`
	Content     string          `json:"content"`
	SourceRunID *uuid.UUID      `json:"source_run_id,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}
