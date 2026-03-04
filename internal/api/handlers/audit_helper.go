package handlers

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	mw "github.com/neuco-ai/neuco/internal/api/middleware"
	"github.com/neuco-ai/neuco/internal/domain"
)

// recordAudit writes an audit log entry for a mutating operation.
// It is fire-and-forget: errors are logged but never surface to the caller.
func recordAudit(
	ctx context.Context,
	d *Deps,
	orgID uuid.UUID,
	action string,
	resource string,
	resourceID string,
	meta any,
) {
	userID := mw.UserIDFromCtx(ctx)

	var raw json.RawMessage
	if meta != nil {
		b, err := json.Marshal(meta)
		if err == nil {
			raw = b
		}
	}

	var resID *uuid.UUID
	if parsed, err := uuid.Parse(resourceID); err == nil {
		resID = &parsed
	}

	entry := domain.AuditEntry{
		ID:         uuid.New(),
		OrgID:      orgID,
		UserID:     &userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resID,
		Metadata:   raw,
	}

	if _, err := d.Store.CreateAuditEntry(ctx, entry); err != nil {
		slog.ErrorContext(ctx, "failed to write audit entry",
			"error", err,
			"action", action,
			"resource", resource,
		)
	}
}
