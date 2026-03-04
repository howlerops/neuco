package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/neuco-ai/neuco/internal/domain"
)

// ──────────────────────────────────────────────────────────────────────────────
// Types
// ──────────────────────────────────────────────────────────────────────────────

// SignalQueryFilters holds optional filter criteria for vector similarity
// searches. It mirrors ai.SignalQueryFilters to avoid a circular import.
type SignalQueryFilters struct {
	Sources []string
	Types   []string
	From    *time.Time
	To      *time.Time
}

// SignalSearchResult is a domain.Signal augmented with a similarity distance
// returned by pgvector. Distance is in [0,2]; lower values mean higher
// similarity (0 = identical vectors).
type SignalSearchResult struct {
	domain.Signal
	// Distance is the cosine distance: 1 - cosine_similarity.
	// Values near 0 indicate high semantic similarity.
	Distance float64 `json:"distance"`
}

// ──────────────────────────────────────────────────────────────────────────────
// SearchSignalsByEmbedding
// ──────────────────────────────────────────────────────────────────────────────

// SearchSignalsByEmbedding performs an approximate nearest-neighbour search
// over signals that have an embedding using the pgvector cosine distance
// operator (<=>). Results are ordered by ascending distance (most similar
// first) and respect any additional filters.
//
// The embedding argument is serialised to a pgvector literal before being
// bound as a parameter, keeping this compatible with vanilla pgx (no
// pgvector-go codec required).
func (s *Store) SearchSignalsByEmbedding(
	ctx context.Context,
	projectID uuid.UUID,
	embedding []float32,
	filters SignalQueryFilters,
	limit int,
) ([]SignalSearchResult, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Serialise the query vector to a pgvector literal "[x,y,…]".
	vectorLit := float32SliceToVectorLiteral(embedding)

	// Build WHERE clause dynamically. $1 = project_id, $2 = vector literal.
	// Additional filter parameters start at $3.
	args := []any{projectID, vectorLit}
	conds := []string{
		"project_id = $1",
		"embedding IS NOT NULL",
	}

	if len(filters.Sources) > 0 {
		args = append(args, filters.Sources)
		conds = append(conds, fmt.Sprintf("source = ANY($%d)", len(args)))
	}
	if len(filters.Types) > 0 {
		args = append(args, filters.Types)
		conds = append(conds, fmt.Sprintf("type = ANY($%d)", len(args)))
	}
	if filters.From != nil {
		args = append(args, *filters.From)
		conds = append(conds, fmt.Sprintf("occurred_at >= $%d", len(args)))
	}
	if filters.To != nil {
		args = append(args, *filters.To)
		conds = append(conds, fmt.Sprintf("occurred_at <= $%d", len(args)))
	}

	args = append(args, limit)
	limitPlaceholder := len(args)

	where := "WHERE " + strings.Join(conds, " AND ")

	// The distance expression references $2 which is the vector literal cast
	// to vector type. pgvector automatically casts string literals to vector
	// when the operator argument type is vector.
	query := fmt.Sprintf(`
		SELECT
			id, project_id, source, source_ref, type, content, metadata,
			occurred_at, ingested_at,
			(embedding <=> $2::vector) AS distance
		FROM   signals
		%s
		ORDER  BY embedding <=> $2::vector
		LIMIT  $%d`,
		where, limitPlaceholder,
	)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store.SearchSignalsByEmbedding: %w", err)
	}
	defer rows.Close()

	var results []SignalSearchResult
	for rows.Next() {
		var r SignalSearchResult
		var meta []byte
		if err := rows.Scan(
			&r.ID,
			&r.ProjectID,
			&r.Source,
			&r.SourceRef,
			&r.Type,
			&r.Content,
			&meta,
			&r.OccurredAt,
			&r.IngestedAt,
			&r.Distance,
		); err != nil {
			return nil, fmt.Errorf("store.SearchSignalsByEmbedding: scan: %w", err)
		}
		if meta != nil {
			r.Metadata = meta
		} else {
			r.Metadata = []byte(`{}`)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.SearchSignalsByEmbedding: rows: %w", err)
	}
	return results, nil
}
