package handlers

import (
	"net/http"
	"strconv"

	mw "github.com/neuco-ai/neuco/internal/api/middleware"
	"github.com/neuco-ai/neuco/internal/store"
)

// AuditLog handles GET /api/v1/orgs/{orgId}/audit-log.
// Returns a paginated, filterable audit log for the org.
// Query params: action, resource, limit, offset.
func AuditLog(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := mw.ResolvedOrgIDFromCtx(r.Context())

		action := r.URL.Query().Get("action")
		resource := r.URL.Query().Get("resource")

		limit := 50
		if lStr := r.URL.Query().Get("limit"); lStr != "" {
			if n, err := strconv.Atoi(lStr); err == nil && n > 0 && n <= 500 {
				limit = n
			}
		}

		offset := 0
		if oStr := r.URL.Query().Get("offset"); oStr != "" {
			if n, err := strconv.Atoi(oStr); err == nil && n >= 0 {
				offset = n
			}
		}

		filters := store.AuditFilters{
			Action:       action,
			ResourceType: resource,
		}

		page, err := d.Store.ListOrgAuditLog(r.Context(), orgID, filters, store.Page(limit, offset))
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to fetch audit log")
			return
		}

		respondOK(w, r, map[string]any{
			"entries": page.Entries,
			"total":   page.Total,
		})
	}
}
