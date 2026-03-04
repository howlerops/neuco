package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/neuco-ai/neuco/internal/domain"
)

const copilotNoteColumns = `
	id, project_id, target_type, target_id, note_type,
	content, metadata, dismissed, created_at`

// CreateCopilotNote inserts a new AI-generated insight note.
func (s *Store) CreateCopilotNote(ctx context.Context, note *domain.CopilotNote) error {
	const q = `
		INSERT INTO copilot_notes (id, project_id, target_type, target_id, note_type, content, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	metadata := note.Metadata
	if metadata == nil {
		metadata = []byte(`{}`)
	}

	_, err := s.pool.Exec(ctx, q,
		note.ID,
		note.ProjectID,
		note.TargetType,
		note.TargetID,
		note.NoteType,
		note.Content,
		metadata,
	)
	if err != nil {
		return fmt.Errorf("store.CreateCopilotNote: %w", err)
	}
	return nil
}

// ListCopilotNotes returns notes for a project, optionally filtered.
func (s *Store) ListCopilotNotes(
	ctx context.Context,
	projectID uuid.UUID,
	targetType domain.CopilotNoteTargetType,
	targetID *uuid.UUID,
	includeDismissed bool,
) ([]domain.CopilotNote, error) {
	args := []any{projectID}
	conds := []string{"project_id = $1"}

	if targetType != "" {
		args = append(args, targetType)
		conds = append(conds, fmt.Sprintf("target_type = $%d", len(args)))
	}
	if targetID != nil {
		args = append(args, *targetID)
		conds = append(conds, fmt.Sprintf("target_id = $%d", len(args)))
	}
	if !includeDismissed {
		conds = append(conds, "dismissed = FALSE")
	}

	where := "WHERE "
	for i, c := range conds {
		if i > 0 {
			where += " AND "
		}
		where += c
	}

	q := "SELECT " + copilotNoteColumns + " FROM copilot_notes " + where + " ORDER BY created_at DESC"

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("store.ListCopilotNotes: %w", err)
	}
	defer rows.Close()

	var notes []domain.CopilotNote
	for rows.Next() {
		note, err := scanCopilotNote(rows)
		if err != nil {
			return nil, fmt.Errorf("store.ListCopilotNotes: scan: %w", err)
		}
		notes = append(notes, note)
	}
	return notes, rows.Err()
}

// DismissCopilotNote marks a note as dismissed.
func (s *Store) DismissCopilotNote(ctx context.Context, projectID, noteID uuid.UUID) error {
	const q = `
		UPDATE copilot_notes
		SET    dismissed = TRUE
		WHERE  id = $1 AND project_id = $2`

	ct, err := s.pool.Exec(ctx, q, noteID, projectID)
	if err != nil {
		return fmt.Errorf("store.DismissCopilotNote: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("store.DismissCopilotNote: note %s not found", noteID)
	}
	return nil
}

func scanCopilotNote(row pgx.Row) (domain.CopilotNote, error) {
	var n domain.CopilotNote
	err := row.Scan(
		&n.ID,
		&n.ProjectID,
		&n.TargetType,
		&n.TargetID,
		&n.NoteType,
		&n.Content,
		&n.Metadata,
		&n.Dismissed,
		&n.CreatedAt,
	)
	return n, err
}
