package codegen

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/neuco-ai/neuco/internal/domain"
)

const (
	AuditActionSandboxProvision  = "sandbox_provision"
	AuditActionSandboxDestroy    = "sandbox_destroy"
	AuditActionAgentStart        = "agent_start"
	AuditActionAgentComplete     = "agent_complete"
	AuditActionAgentFail         = "agent_fail"
	AuditActionValidationStart   = "validation_start"
	AuditActionValidationPass    = "validation_pass"
	AuditActionValidationFail    = "validation_fail"
	AuditActionRetryAttempt      = "retry_attempt"
	AuditActionGenerationStart   = "generation_start"
	AuditActionGenerationComplete = "generation_complete"
	AuditActionGenerationFail    = "generation_fail"
	AuditActionConfigUpdate      = "config_update"
	AuditActionConfigDelete      = "config_delete"
)

// AuditStore defines the audit persistence dependency for codegen flows.
type AuditStore interface {
	CreateAuditEntry(ctx context.Context, entry domain.AuditEntry) (domain.AuditEntry, error)
}

// RecordAudit writes a codegen audit log entry.
// It is fire-and-forget: errors are logged, never returned to the caller.
func RecordAudit(
	ctx context.Context,
	store AuditStore,
	orgID uuid.UUID,
	action string,
	resource string,
	resourceID uuid.UUID,
	meta map[string]string,
) {
	if store == nil {
		slog.WarnContext(ctx, "record audit skipped: store is nil",
			"action", action,
			"resource", resource,
		)
		return
	}

	rawMeta := json.RawMessage(`{}`)
	if meta != nil {
		b, err := json.Marshal(meta)
		if err != nil {
			slog.WarnContext(ctx, "record audit: failed to marshal metadata",
				"error", err,
				"action", action,
				"resource", resource,
			)
		} else {
			rawMeta = b
		}
	}

	var rid *uuid.UUID
	if resourceID != uuid.Nil {
		rid = &resourceID
	}

	entry := domain.AuditEntry{
		ID:         uuid.New(),
		OrgID:      orgID,
		Action:     action,
		Resource:   resource,
		ResourceID: rid,
		Metadata:   rawMeta,
	}

	if _, err := store.CreateAuditEntry(ctx, entry); err != nil {
		slog.WarnContext(ctx, "record audit: failed to create audit entry",
			"error", err,
			"action", action,
			"resource", resource,
		)
	}
}
