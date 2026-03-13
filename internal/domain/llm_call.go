package domain

import (
	"time"

	"github.com/google/uuid"
)

// LLMCallType identifies the purpose of an LLM call.
type LLMCallType string

const (
	LLMCallTypeSpecGen       LLMCallType = "spec_gen"
	LLMCallTypeCodegen       LLMCallType = "codegen"
	LLMCallTypeThemeNaming   LLMCallType = "theme_naming"
	LLMCallTypeCopilotReview LLMCallType = "copilot_review"
	LLMCallTypeEmbedding      LLMCallType = "embedding"
	LLMCallTypeContextUpdate  LLMCallType = "context_update"
)

// LLMProvider identifies the AI provider.
type LLMProvider string

const (
	LLMProviderAnthropic LLMProvider = "anthropic"
	LLMProviderOpenAI    LLMProvider = "openai"
)

// LLMCall records a single call to an LLM API with token usage and cost.
type LLMCall struct {
	ID             uuid.UUID   `json:"id"`
	ProjectID      uuid.UUID   `json:"project_id"`
	PipelineRunID  *uuid.UUID  `json:"pipeline_run_id,omitempty"`
	PipelineTaskID *uuid.UUID  `json:"pipeline_task_id,omitempty"`
	Provider       LLMProvider `json:"provider"`
	Model          string      `json:"model"`
	CallType       LLMCallType `json:"call_type"`
	TokensIn       int         `json:"tokens_in"`
	TokensOut      int         `json:"tokens_out"`
	LatencyMs      int         `json:"latency_ms"`
	CostUSD        float64     `json:"cost_usd"`
	ErrorMsg       string      `json:"error,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
}

// LLMUsageAgg holds aggregated LLM usage stats for a pipeline run or project.
type LLMUsageAgg struct {
	TotalCalls    int     `json:"total_calls"`
	TotalTokensIn int     `json:"total_tokens_in"`
	TotalTokensOut int    `json:"total_tokens_out"`
	TotalCostUSD  float64 `json:"total_cost_usd"`
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
	P95LatencyMs  float64 `json:"p95_latency_ms"`
}

// CostPerMillionTokens returns the per-million-token cost for a model.
// Prices in USD per million tokens (input, output).
func CostPerMillionTokens(model string) (inputCost, outputCost float64) {
	switch model {
	case "claude-sonnet-4-6-20250514", "claude-sonnet-4-5":
		return 3.0, 15.0
	case "claude-haiku-4-5-20251001":
		return 0.80, 4.0
	case "text-embedding-3-small":
		return 0.02, 0.0
	default:
		return 3.0, 15.0 // default to Sonnet pricing
	}
}

// CalculateCostUSD computes the USD cost for a call given model and token counts.
func CalculateCostUSD(model string, tokensIn, tokensOut int) float64 {
	inCost, outCost := CostPerMillionTokens(model)
	return (float64(tokensIn) * inCost / 1_000_000) + (float64(tokensOut) * outCost / 1_000_000)
}
