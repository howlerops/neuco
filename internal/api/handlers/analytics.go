package handlers

import (
	"net/http"
	"strconv"

	mw "github.com/neuco-ai/neuco/internal/api/middleware"
)

// GetOrgAnalytics handles GET /api/v1/orgs/{orgId}/analytics.
// Query params:
//   - days: number of days to look back (7, 30, or 90; default 30)
func GetOrgAnalytics(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := mw.ResolvedOrgIDFromCtx(r.Context())

		days := 30
		if dStr := r.URL.Query().Get("days"); dStr != "" {
			if n, err := strconv.Atoi(dStr); err == nil {
				switch n {
				case 7, 30, 90:
					days = n
				}
			}
		}

		analytics, err := d.Store.GetOrgAnalytics(r.Context(), orgID, days)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to get analytics")
			return
		}

		respondOK(w, r, analytics)
	}
}
