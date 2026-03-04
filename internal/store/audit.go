package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/neuco-ai/neuco/internal/domain"
)

// AuditFilters carries optional constraints for ListOrgAuditLog.
type AuditFilters struct {
	Action       string     // e.g. "create_project"; empty = no filter
	ResourceType string     // e.g. "project"; empty = no filter
	ActorID      *uuid.UUID // user_id filter
}

// AuditPage is the result type for paginated audit log queries.
type AuditPage struct {
	Entries []domain.AuditEntry
	Total   int
}

// Column list matching the actual DB schema: user_id (not actor_id), resource (not resource_type)
const auditColumns = `
	id, org_id, user_id, action, resource, resource_id, metadata, created_at`

// CreateAuditEntry inserts an immutable audit log record.
func (s *Store) CreateAuditEntry(ctx context.Context, entry domain.AuditEntry) (domain.AuditEntry, error) {
	meta := entry.Metadata
	if meta == nil {
		meta = json.RawMessage(`{}`)
	}

	const q = `
		INSERT INTO audit_log
		       (org_id, user_id, action, resource, resource_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING ` + auditColumns

	row := s.pool.QueryRow(ctx, q,
		entry.OrgID,
		entry.UserID,
		entry.Action,
		entry.Resource,
		entry.ResourceID,
		meta,
	)
	out, err := scanAuditEntry(row)
	if err != nil {
		return domain.AuditEntry{}, fmt.Errorf("store.CreateAuditEntry: %w", err)
	}
	return out, nil
}

// ListOrgAuditLog returns a paginated audit log for an org.
func (s *Store) ListOrgAuditLog(ctx context.Context, orgID uuid.UUID, filters AuditFilters, pp PageParams) (AuditPage, error) {
	args := []any{orgID}
	conds := []string{"org_id = $1"}

	if filters.Action != "" {
		args = append(args, filters.Action)
		conds = append(conds, fmt.Sprintf("action = $%d", len(args)))
	}
	if filters.ResourceType != "" {
		args = append(args, filters.ResourceType)
		conds = append(conds, fmt.Sprintf("resource = $%d", len(args)))
	}
	if filters.ActorID != nil {
		args = append(args, *filters.ActorID)
		conds = append(conds, fmt.Sprintf("user_id = $%d", len(args)))
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	countQ := "SELECT COUNT(*) FROM audit_log " + where
	var total int
	if err := s.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return AuditPage{}, fmt.Errorf("store.ListOrgAuditLog count: %w", err)
	}

	args = append(args, pp.Limit, pp.Offset)
	dataQ := fmt.Sprintf(
		"SELECT %s FROM audit_log %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		auditColumns, where, len(args)-1, len(args),
	)

	rows, err := s.pool.Query(ctx, dataQ, args...)
	if err != nil {
		return AuditPage{}, fmt.Errorf("store.ListOrgAuditLog: %w", err)
	}
	defer rows.Close()

	var entries []domain.AuditEntry
	for rows.Next() {
		entry, err := scanAuditEntry(rows)
		if err != nil {
			return AuditPage{}, fmt.Errorf("store.ListOrgAuditLog: scan: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return AuditPage{}, fmt.Errorf("store.ListOrgAuditLog: rows: %w", err)
	}
	return AuditPage{Entries: entries, Total: total}, nil
}

func scanAuditEntry(row pgx.Row) (domain.AuditEntry, error) {
	var e domain.AuditEntry
	var meta []byte
	err := row.Scan(
		&e.ID,
		&e.OrgID,
		&e.UserID,
		&e.Action,
		&e.Resource,
		&e.ResourceID,
		&meta,
		&e.CreatedAt,
	)
	if err != nil {
		return domain.AuditEntry{}, err
	}
	if meta != nil {
		e.Metadata = json.RawMessage(meta)
	} else {
		e.Metadata = json.RawMessage(`{}`)
	}
	return e, nil
}
