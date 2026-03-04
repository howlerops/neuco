// Package agents implements ReAct-style LLM agents for Neuco's AI layer.
// The TranscriptAgent extracts product signals from long call transcripts or
// support threads using a tool-loop rather than sending the full text to the
// LLM in a single prompt. This keeps token usage manageable and lets the model
// focus its attention on the most relevant sections.
package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/neuco-ai/neuco/internal/ai"
	"github.com/neuco-ai/neuco/internal/domain"
	"github.com/neuco-ai/neuco/internal/store"
)

// ──────────────────────────────────────────────────────────────────────────────
// Constants & prompts
// ──────────────────────────────────────────────────────────────────────────────

const (
	maxAgentIterations = 40
	agentModel         = "claude-sonnet-4-5"
)

// systemPrompt is the agent's persona and strategy, exactly as described in the
// architecture document.
const systemPrompt = `You are a product signal extractor. Read a call transcript or support thread and extract discrete product signals.

The full transcript is available in memory. Use your tools to explore it:
- peek(start, end) — read a section by line numbers
- search(pattern) — regex search for keywords across the full text
- sub_query(question, excerpt) — ask a focused question about a small excerpt
- emit_signal(content, type, metadata) — record a signal when you find one

Signal types: feature_request | pain_point | praise | bug_report | question

A good signal is specific, grounded in the speaker's words, and actionable.

Strategy:
1. Peek at the opening and closing (often most signal-dense)
2. Search for: "wish", "want", "can't", "need", "broken", "love", "hate", "always"
3. For each hit, sub_query to get full context, then emit_signal if it qualifies
4. Declare done when you've covered the document`

// ──────────────────────────────────────────────────────────────────────────────
// Tool schemas
// ──────────────────────────────────────────────────────────────────────────────

var agentTools = []ai.ToolDef{
	{
		Name:        "peek",
		Description: "Read a section of the transcript by 0-based line numbers (inclusive on both ends).",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"start": map[string]interface{}{"type": "integer", "description": "First line to return (0-based)"},
				"end":   map[string]interface{}{"type": "integer", "description": "Last line to return (inclusive)"},
			},
			"required": []string{"start", "end"},
		},
	},
	{
		Name:        "search",
		Description: "Regex search across the full transcript. Returns matching lines with their line numbers.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{"type": "string", "description": "Go-compatible regex pattern"},
			},
			"required": []string{"pattern"},
		},
	},
	{
		Name:        "sub_query",
		Description: "Ask a focused natural-language question about a specific excerpt. Returns a concise answer.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"question": map[string]interface{}{"type": "string", "description": "Focused question about the excerpt"},
				"excerpt":  map[string]interface{}{"type": "string", "description": "The transcript excerpt to analyse"},
			},
			"required": []string{"question", "excerpt"},
		},
	},
	{
		Name:        "emit_signal",
		Description: "Record an extracted product signal. Call this once per distinct signal you find.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"content":     map[string]interface{}{"type": "string", "description": "The signal text, ideally a direct quote or close paraphrase"},
				"signal_type": map[string]interface{}{"type": "string", "enum": []string{"feature_request", "pain_point", "praise", "bug_report", "question"}},
				"metadata": map[string]interface{}{
					"type":                 "object",
					"additionalProperties": map[string]interface{}{"type": "string"},
					"description":          "Optional string key/value pairs, e.g. speaker name, timestamp",
				},
			},
			"required": []string{"content", "signal_type"},
		},
	},
}

// ──────────────────────────────────────────────────────────────────────────────
// TranscriptAgent
// ──────────────────────────────────────────────────────────────────────────────

// TranscriptAgent is a ReAct agent that extracts product signals from a long
// transcript without ever sending the full text to the LLM in a single call.
// It stores the transcript in a Go variable and exposes it to the model via
// tools (peek, search, sub_query). Extracted signals are accumulated locally
// and returned to the caller.
type TranscriptAgent struct {
	llm   *ai.LLMClient
	store *store.Store
}

// NewTranscriptAgent constructs an agent. The store is used to persist emitted
// signals as they are found so that partial results survive if the agent loop
// is interrupted.
func NewTranscriptAgent(llm *ai.LLMClient, s *store.Store) *TranscriptAgent {
	return &TranscriptAgent{llm: llm, store: s}
}

// Process runs the agent loop against transcript and returns all extracted
// signals. If the projectID is non-nil the signals are also persisted via the
// store's InsertSignal method so they are immediately queryable.
func (a *TranscriptAgent) Process(ctx context.Context, projectID uuid.UUID, transcript string) ([]domain.Signal, error) {
	lines := strings.Split(transcript, "\n")

	// agentState holds mutable state that the tool implementations close over.
	state := &agentState{
		lines:     lines,
		llm:       a.llm,
		store:     a.store,
		projectID: projectID,
	}

	// Build the initial user message.
	initialUser := fmt.Sprintf(
		"The transcript has %d lines. Begin by peeking at the first 30 lines and the last 30 lines, then proceed with your strategy.",
		len(lines),
	)

	// Conversation history in Anthropic's multi-turn format. We use a raw
	// []map[string]interface{} slice because tool results require the
	// "tool_result" content-block structure that our simple ai.Message type
	// does not express. We translate to []ai.Message for the API call by
	// marshalling to JSON and back — but we keep the raw form here for easy
	// manipulation.
	type rawMessage = map[string]interface{}
	history := []rawMessage{
		{"role": "user", "content": initialUser},
	}

	for iter := 0; iter < maxAgentIterations; iter++ {
		// Convert raw history to []ai.Message for the LLMClient call.
		// We serialise each entry's content through JSON so we can handle both
		// plain string content and content-block arrays.
		apiMessages, err := rawToAPIMessages(history)
		if err != nil {
			return state.signals, fmt.Errorf("transcript_agent: marshal messages: %w", err)
		}

		resp, err := a.llm.ChatWithTools(ctx, agentModel, systemPrompt, apiMessages, agentTools, 4096)
		if err != nil {
			return state.signals, fmt.Errorf("transcript_agent: iteration %d: %w", iter, err)
		}

		slog.Debug("transcript_agent: iteration", "iter", iter, "stop_reason", resp.StopReason, "tool_calls", len(resp.ToolCalls))

		if resp.StopReason == "end_turn" || len(resp.ToolCalls) == 0 {
			// The model signalled it is done.
			break
		}

		// Build the assistant turn from the raw response body. We need to
		// reconstruct the full content-block array so the API receives it in
		// subsequent turns.
		assistantContent := buildAssistantContent(resp)
		history = append(history, rawMessage{
			"role":    "assistant",
			"content": assistantContent,
		})

		// Execute every tool call and collect results for the "user" turn.
		toolResultBlocks := make([]interface{}, 0, len(resp.ToolCalls))
		for _, tc := range resp.ToolCalls {
			result := state.executeTool(ctx, tc)
			toolResultBlocks = append(toolResultBlocks, map[string]interface{}{
				"type":        "tool_result",
				"tool_use_id": tc.ID,
				"content":     result,
			})
		}

		history = append(history, rawMessage{
			"role":    "user",
			"content": toolResultBlocks,
		})
	}

	return state.signals, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Agent state & tool execution
// ──────────────────────────────────────────────────────────────────────────────

// agentState holds the in-memory transcript and accumulates emitted signals.
type agentState struct {
	lines     []string
	llm       *ai.LLMClient
	store     *store.Store
	projectID uuid.UUID
	signals   []domain.Signal
}

// executeTool dispatches a ToolCall to the appropriate implementation.
func (s *agentState) executeTool(ctx context.Context, tc ai.ToolCall) string {
	switch tc.Name {
	case "peek":
		return s.toolPeek(tc.Input)
	case "search":
		return s.toolSearch(tc.Input)
	case "sub_query":
		return s.toolSubQuery(ctx, tc.Input)
	case "emit_signal":
		return s.toolEmitSignal(ctx, tc.Input)
	default:
		return fmt.Sprintf("unknown tool: %s", tc.Name)
	}
}

// toolPeek returns lines [start, end] (0-based, inclusive).
func (s *agentState) toolPeek(raw json.RawMessage) string {
	var args struct {
		Start int `json:"start"`
		End   int `json:"end"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return fmt.Sprintf("peek error: bad args: %s", err)
	}

	total := len(s.lines)
	start := clamp(args.Start, 0, total-1)
	end := clamp(args.End, start, total-1)

	var sb strings.Builder
	for i := start; i <= end; i++ {
		fmt.Fprintf(&sb, "%d: %s\n", i, s.lines[i])
	}
	return sb.String()
}

// searchMatch is a single result from toolSearch.
type searchMatch struct {
	Line    int    `json:"line"`
	Content string `json:"content"`
}

// toolSearch performs a regex search across all lines and returns matching
// lines with line numbers as a JSON array.
func (s *agentState) toolSearch(raw json.RawMessage) string {
	var args struct {
		Pattern string `json:"pattern"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return fmt.Sprintf("search error: bad args: %s", err)
	}

	re, err := regexp.Compile("(?i)" + args.Pattern)
	if err != nil {
		return fmt.Sprintf("search error: invalid pattern %q: %s", args.Pattern, err)
	}

	var matches []searchMatch
	for i, line := range s.lines {
		if re.MatchString(line) {
			matches = append(matches, searchMatch{Line: i, Content: line})
		}
	}

	if len(matches) == 0 {
		return "no matches"
	}

	// Cap to 50 matches to keep the context window manageable.
	if len(matches) > 50 {
		matches = matches[:50]
	}

	out, _ := json.MarshalIndent(matches, "", "  ")
	return string(out)
}

// toolSubQuery sends a focused prompt to Claude Haiku about the provided
// excerpt and returns its answer.
func (s *agentState) toolSubQuery(ctx context.Context, raw json.RawMessage) string {
	var args struct {
		Question string `json:"question"`
		Excerpt  string `json:"excerpt"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return fmt.Sprintf("sub_query error: bad args: %s", err)
	}

	const subQuerySystem = "You are a precise research assistant. Answer the question based solely on the provided excerpt. Be concise."
	userPrompt := fmt.Sprintf("Excerpt:\n%s\n\nQuestion: %s", args.Excerpt, args.Question)

	answer, err := s.llm.ChatHaiku(ctx, subQuerySystem, userPrompt, 512)
	if err != nil {
		return fmt.Sprintf("sub_query error: %s", err)
	}
	return answer
}

// toolEmitSignal constructs a domain.Signal, appends it to the in-memory
// slice, persists it via the store, and returns a confirmation string.
func (s *agentState) toolEmitSignal(ctx context.Context, raw json.RawMessage) string {
	var args struct {
		Content    string            `json:"content"`
		SignalType string            `json:"signal_type"`
		Metadata   map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return fmt.Sprintf("emit_signal error: bad args: %s", err)
	}

	if args.Content == "" {
		return "emit_signal error: content is required"
	}

	// Map agent signal types to domain.SignalType constants.
	signalType := mapSignalType(args.SignalType)

	meta := map[string]string{"agent": "transcript"}
	for k, v := range args.Metadata {
		meta[k] = v
	}
	metaJSON, _ := json.Marshal(meta)

	sig := domain.Signal{
		ID:         uuid.New(),
		ProjectID:  s.projectID,
		Source:     domain.SignalSourceManual,
		Type:       signalType,
		Content:    args.Content,
		Metadata:   metaJSON,
		OccurredAt: time.Now().UTC(),
	}

	// Persist immediately so partial results survive worker interruption.
	if s.store != nil && s.projectID != uuid.Nil {
		inserted, err := s.store.InsertSignal(ctx, sig)
		if err != nil {
			slog.Error("transcript_agent: failed to persist signal", "error", err)
			// Append in-memory even if persistence failed.
			s.signals = append(s.signals, sig)
			return fmt.Sprintf("emit_signal: persisted failed (%s), buffered in-memory: %s", err, sig.ID)
		}
		s.signals = append(s.signals, inserted)
		return fmt.Sprintf("signal recorded: id=%s type=%s", inserted.ID, inserted.Type)
	}

	s.signals = append(s.signals, sig)
	return fmt.Sprintf("signal buffered: id=%s type=%s", sig.ID, sig.Type)
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────────

// mapSignalType converts agent-facing type names to domain.SignalType.
func mapSignalType(t string) domain.SignalType {
	switch strings.ToLower(t) {
	case "feature_request":
		return domain.SignalTypeFeatureRequest
	case "pain_point":
		return domain.SignalTypeNote // closest domain type for general pain
	case "praise":
		return domain.SignalTypeProductReview
	case "bug_report":
		return domain.SignalTypeBugReport
	case "question":
		return domain.SignalTypeNote
	default:
		return domain.SignalTypeNote
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// rawToAPIMessages converts our raw map history into the []ai.Message slice
// the LLMClient expects. Messages whose content is already a plain string pass
// through directly; those with structured content blocks are JSON-re-encoded
// so the LLM sees them correctly.
//
// Note: The Anthropic API actually accepts "content" as either a string or a
// content-block array. Our LLMClient sends messages as-is through
// json.Marshal, so we store pre-marshalled content blocks as strings that
// will be decoded back to their array form by the API serialisation layer.
// To handle this cleanly we marshal the entire message list and let the client
// decode it. We embed the raw content directly rather than round-tripping
// through a string.
func rawToAPIMessages(history []map[string]interface{}) ([]ai.Message, error) {
	// We cannot losslessly represent structured content blocks in ai.Message
	// (which only has a string Content field). Instead we serialise the raw
	// history to JSON, inject it as a single batch-send inside ChatWithTools,
	// and the internal doAnthropic call will include it verbatim.
	//
	// Since ChatWithTools builds its apiMessages slice from []ai.Message, we
	// need to special-case this. The trick we use: serialise any non-string
	// content to a JSON string so it round-trips through the content field.
	// The Anthropic API's messages endpoint accepts a "content" field that is
	// either a string or an array of content blocks.
	//
	// Our doWithRetry path calls json.Marshal(payload) where payload.messages
	// is []map[string]interface{}, so we can pass the raw history directly to
	// a variant of the chat call. We expose this by passing the raw maps as
	// ai.Message{Content: "<json>"} and using a special sentinal role.
	//
	// Simpler approach: the caller already passes []ai.Message to
	// ChatWithTools; we construct them here by encoding non-string content to
	// a JSON string for the Content field. Claude's messages API accepts
	// string content directly.

	out := make([]ai.Message, 0, len(history))
	for _, m := range history {
		role, _ := m["role"].(string)
		switch c := m["content"].(type) {
		case string:
			out = append(out, ai.Message{Role: role, Content: c})
		default:
			// Marshal content blocks to JSON string.
			encoded, err := json.Marshal(c)
			if err != nil {
				return nil, err
			}
			out = append(out, ai.Message{Role: role, Content: string(encoded)})
		}
	}
	return out, nil
}

// buildAssistantContent reconstructs the full Anthropic content-block array
// from a ToolCallResponse so it can be stored in the conversation history and
// re-submitted in subsequent turns. The API requires that the assistant's turn
// includes the exact tool_use blocks it generated.
func buildAssistantContent(resp *ai.ToolCallResponse) []interface{} {
	var blocks []interface{}

	if resp.Content != "" {
		blocks = append(blocks, map[string]interface{}{
			"type": "text",
			"text": resp.Content,
		})
	}

	for _, tc := range resp.ToolCalls {
		var input interface{}
		_ = json.Unmarshal(tc.Input, &input)
		blocks = append(blocks, map[string]interface{}{
			"type":  "tool_use",
			"id":    tc.ID,
			"name":  tc.Name,
			"input": input,
		})
	}

	return blocks
}
