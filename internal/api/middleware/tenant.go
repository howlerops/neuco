package middleware

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/neuco-ai/neuco/internal/store"
)

// ProjectTenant extracts {projectId} from the URL, verifies the project
// belongs to the org present in the JWT (set by Authenticate), and stores the
// ProjectID in the request context. Responds 404 if the project is not found
// or does not belong to the org.
func ProjectTenant(s *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := chi.URLParam(r, "projectId")
			projectID, err := uuid.Parse(raw)
			if err != nil {
				http.Error(w, `{"error":"invalid project_id"}`, http.StatusBadRequest)
				return
			}

			orgID := OrgIDFromCtx(r.Context())
			if orgID == uuid.Nil {
				http.Error(w, `{"error":"missing org context"}`, http.StatusUnauthorized)
				return
			}

			// GetProject(ctx, orgID, projectID) — the org scope enforces tenant isolation.
			_, err = s.GetProject(r.Context(), orgID, projectID)
			if err != nil {
				// Do not reveal whether the project exists for a different org.
				http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
				return
			}

			ctx := context.WithValue(r.Context(), ctxKeyProjectID, projectID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ProjectIDFromCtx retrieves the verified ProjectID from the context.
// Returns uuid.Nil if the tenant middleware has not run.
func ProjectIDFromCtx(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(ctxKeyProjectID).(uuid.UUID)
	return v
}
