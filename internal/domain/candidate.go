package domain

import (
	"time"

	"github.com/google/uuid"
)

// CandidateStatus tracks where a feature candidate is in the product workflow.
type CandidateStatus string

const (
	CandidateStatusNew        CandidateStatus = "new"
	CandidateStatusSpecced    CandidateStatus = "specced"
	CandidateStatusInProgress CandidateStatus = "in_progress"
	CandidateStatusReviewing  CandidateStatus = "reviewing"
	CandidateStatusAccepted   CandidateStatus = "accepted"
	CandidateStatusRejected   CandidateStatus = "rejected"
	CandidateStatusBacklogged CandidateStatus = "backlogged"
	CandidateStatusShipped    CandidateStatus = "shipped"
)

// FeatureCandidate is a synthesised product opportunity surfaced from clustered
// signals. Candidates are scored and ranked so the most impactful ones appear
// first. Each candidate may have a Spec generated from it.
type FeatureCandidate struct {
	ID               uuid.UUID       `json:"id"`
	ProjectID        uuid.UUID       `json:"project_id"`
	Title            string          `json:"title"`
	ProblemSummary   string          `json:"problem_summary"`
	Status           CandidateStatus `json:"status"`
	Score            float64         `json:"score"`
	SignalCount      int             `json:"signal_count"`
	FrequencyScore   float64         `json:"frequency_score"`
	RecencyScore     float64         `json:"recency_score"`
	SegmentWeight    float64         `json:"segment_weight"`
	ChurnRiskScore   float64         `json:"churn_risk_score"`
	ClusterID        string          `json:"cluster_id"`
	SynthesisRunID   *uuid.UUID      `json:"synthesis_run_id,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// CandidateSignal is the join between a FeatureCandidate and the Signals that
// support it, along with the semantic similarity score used during clustering.
type CandidateSignal struct {
	CandidateID     uuid.UUID `json:"candidate_id"`
	SignalID         uuid.UUID `json:"signal_id"`
	SimilarityScore  float64   `json:"similarity_score"`
	IsRepresentative bool      `json:"is_representative"`
}
