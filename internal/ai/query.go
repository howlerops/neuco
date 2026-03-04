package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/neuco-ai/neuco/internal/store"
)

// ──────────────────────────────────────────────────────────────────────────────
// SignalQueryEngine
// ──────────────────────────────────────────────────────────────────────────────

// SignalQueryFilters holds optional filter criteria for a semantic signal query.
type SignalQueryFilters struct {
	// Sources restricts results to the given signal sources (e.g. "gong").
	Sources []string
	// Types restricts results to the given signal types.
	Types []string
	// From is an inclusive lower bound on occurred_at.
	From *time.Time
	// To is an inclusive upper bound on occurred_at.
	To *time.Time
	// Limit caps the number of results (default 20, maximum 100).
	Limit int
}

// QueryResult pairs a domain signal with its cosine similarity distance.
// A smaller Distance means higher semantic similarity.
type QueryResult struct {
	store.SignalSearchResult
}

// SignalQueryEngine answers natural-language questions about signals using
// pgvector nearest-neighbour search. Questions are embedded via the LLMClient
// and compared against signal embeddings stored in the database.
type SignalQueryEngine struct {
	llm   *LLMClient
	store SignalSearcher
}

// SignalSearcher is the subset of *store.Store used by SignalQueryEngine. It
// is defined as an interface so the engine can be unit-tested with a stub.
type SignalSearcher interface {
	SearchSignalsByEmbedding(
		ctx context.Context,
		projectID uuid.UUID,
		embedding []float32,
		filters store.SignalQueryFilters,
		limit int,
	) ([]store.SignalSearchResult, error)
}

// NewSignalQueryEngine constructs the engine.
func NewSignalQueryEngine(llm *LLMClient, s SignalSearcher) *SignalQueryEngine {
	return &SignalQueryEngine{llm: llm, store: s}
}

// Query embeds question, executes a vector similarity search, and returns the
// top-N matching signals with their similarity scores.
func (e *SignalQueryEngine) Query(
	ctx context.Context,
	projectID uuid.UUID,
	question string,
	filters SignalQueryFilters,
) ([]QueryResult, error) {
	if question == "" {
		return nil, fmt.Errorf("ai.SignalQueryEngine.Query: question must not be empty")
	}

	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// 1. Embed the question.
	embedding, err := e.llm.GenerateEmbedding(ctx, question)
	if err != nil {
		return nil, fmt.Errorf("ai.SignalQueryEngine.Query: embed question: %w", err)
	}

	// 2. Translate our filter type into the store's filter type.
	storeFilters := store.SignalQueryFilters{
		Sources: filters.Sources,
		Types:   filters.Types,
		From:    filters.From,
		To:      filters.To,
	}

	// 3. Search the database.
	rows, err := e.store.SearchSignalsByEmbedding(ctx, projectID, embedding, storeFilters, limit)
	if err != nil {
		return nil, fmt.Errorf("ai.SignalQueryEngine.Query: vector search: %w", err)
	}

	results := make([]QueryResult, len(rows))
	for i, r := range rows {
		results[i] = QueryResult{SignalSearchResult: r}
	}
	return results, nil
}
