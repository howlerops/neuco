package middleware

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/neuco-ai/neuco/internal/store"
)

type ctxKeyResolvedOrgID struct{}

// ResolveOrg is middleware that resolves the {orgId} URL parameter — which may
// be either a UUID or a slug — into a validated UUID and stores it in context.
// Handlers downstream call ResolvedOrgIDFromCtx to get the UUID.
func ResolveOrg(s *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			param := chi.URLParam(r, "orgId")

			// Try UUID first
			orgID, err := uuid.Parse(param)
			if err != nil {
				// Not a UUID — look up by slug
				org, err := s.GetOrgBySlug(r.Context(), param)
				if err != nil {
					http.Error(w, `{"error":"org not found"}`, http.StatusNotFound)
					return
				}
				orgID = org.ID
			}

			ctx := context.WithValue(r.Context(), ctxKeyResolvedOrgID{}, orgID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ResolvedOrgIDFromCtx returns the org UUID resolved by ResolveOrg middleware.
func ResolvedOrgIDFromCtx(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(ctxKeyResolvedOrgID{}).(uuid.UUID)
	return id
}
