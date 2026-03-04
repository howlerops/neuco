package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SignalSource identifies where a signal originated.
type SignalSource string

const (
	SignalSourceGong      SignalSource = "gong"
	SignalSourceIntercom  SignalSource = "intercom"
	SignalSourceLinear    SignalSource = "linear"
	SignalSourceGitHub    SignalSource = "github"
	SignalSourceSlack     SignalSource = "slack"
	SignalSourceZendesk   SignalSource = "zendesk"
	SignalSourceHubSpot   SignalSource = "hubspot"
	SignalSourceAmplitude SignalSource = "amplitude"
	SignalSourceMixpanel  SignalSource = "mixpanel"
	SignalSourceCSV       SignalSource = "csv"
	SignalSourceManual    SignalSource = "manual"
	SignalSourceWebhook   SignalSource = "webhook"
)

// SignalType classifies the kind of customer feedback or product signal.
type SignalType string

const (
	SignalTypeCallTranscript  SignalType = "call_transcript"
	SignalTypeSupportTicket   SignalType = "support_ticket"
	SignalTypeFeatureRequest  SignalType = "feature_request"
	SignalTypeBugReport       SignalType = "bug_report"
	SignalTypeUserInterview    SignalType = "user_interview"
	SignalTypeSurveyResponse  SignalType = "survey_response"
	SignalTypeNPSComment      SignalType = "nps_comment"
	SignalTypeChurnReason     SignalType = "churn_reason"
	SignalTypeProductReview   SignalType = "product_review"
	SignalTypeSlackMessage    SignalType = "slack_message"
	SignalTypeGitHubIssue     SignalType = "github_issue"
	SignalTypeLinearIssue     SignalType = "linear_issue"
	SignalTypeUsageAnomaly    SignalType = "usage_anomaly"
	SignalTypeNote            SignalType = "note"
	SignalTypeEvent           SignalType = "event"
	SignalTypeReview          SignalType = "review"
)

// Signal is a single unit of customer feedback or product insight ingested
// into a project. Signals are embedded and clustered to surface candidates.
type Signal struct {
	ID         uuid.UUID       `json:"id"`
	ProjectID  uuid.UUID       `json:"project_id"`
	Source     SignalSource    `json:"source"`
	SourceRef  string          `json:"source_ref"`
	Type       SignalType      `json:"type"`
	Content    string          `json:"content"`
	Metadata   json.RawMessage `json:"metadata"`
	OccurredAt time.Time       `json:"occurred_at"`
	IngestedAt time.Time       `json:"ingested_at"`

	// Embedding is stored as a pgvector vector column. It is nil until the
	// background embedder worker processes the signal.
	Embedding []float32 `json:"embedding,omitempty"`
}
