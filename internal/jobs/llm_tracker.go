package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/neuco-ai/neuco/internal/domain"
	"github.com/neuco-ai/neuco/internal/store"
)

// anthropicUsage maps the usage object from the Anthropic Messages API response.
type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// anthropicResponse is the full response from the Anthropic Messages API,
// including the usage field needed for tracking.
type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Usage anthropicUsage `json:"usage"`
	Model string         `json:"model"`
}

// recordLLMCall persists an LLM call record. Errors are logged but not returned
// to avoid disrupting the main job flow.
func recordLLMCall(
	ctx context.Context,
	s *store.Store,
	projectID uuid.UUID,
	runID *uuid.UUID,
	taskID *uuid.UUID,
	provider domain.LLMProvider,
	model string,
	callType domain.LLMCallType,
	tokensIn, tokensOut, latencyMs int,
	errMsg string,
) {
	cost := domain.CalculateCostUSD(model, tokensIn, tokensOut)

	call := &domain.LLMCall{
		ID:             uuid.New(),
		ProjectID:      projectID,
		PipelineRunID:  runID,
		PipelineTaskID: taskID,
		Provider:       provider,
		Model:          model,
		CallType:       callType,
		TokensIn:       tokensIn,
		TokensOut:      tokensOut,
		LatencyMs:      latencyMs,
		CostUSD:        cost,
		ErrorMsg:       errMsg,
	}

	if err := s.CreateLLMCall(ctx, call); err != nil {
		slog.Error("failed to record LLM call",
			"error", err,
			"model", model,
			"call_type", callType,
		)
	} else {
		slog.Info("recorded LLM call",
			"model", model,
			"call_type", callType,
			"tokens_in", tokensIn,
			"tokens_out", tokensOut,
			"latency_ms", latencyMs,
			"cost_usd", cost,
		)
	}
}

// ptrUUID returns a pointer to the UUID, or nil if it's the zero value.
func ptrUUID(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}

// trackDuration returns the elapsed milliseconds since start.
func trackDuration(start time.Time) int {
	return int(time.Since(start).Milliseconds())
}
