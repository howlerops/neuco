package middleware

import (
	"net/http"

	"github.com/neuco-ai/neuco/internal/domain"
)

// roleRank maps each OrgRole to a numeric rank so that hierarchy comparisons
// can be expressed as simple integer inequalities (higher == more privileged).
var roleRank = map[domain.OrgRole]int{
	domain.OrgRoleViewer: 1,
	domain.OrgRoleMember: 2,
	domain.OrgRoleAdmin:  3,
	domain.OrgRoleOwner:  4,
}

// RequireRole returns a middleware that allows the request to proceed only when
// the caller's role (from the JWT context) is greater than or equal to minRole
// in the role hierarchy: owner > admin > member > viewer.
// Responds 403 Forbidden for insufficient privileges and 401 Unauthorized when
// no role is present in the context.
func RequireRole(minRole domain.OrgRole) func(http.Handler) http.Handler {
	minRank := roleRank[minRole]
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := RoleFromCtx(r.Context())
			if role == "" {
				http.Error(w, `{"error":"unauthenticated"}`, http.StatusUnauthorized)
				return
			}
			rank, ok := roleRank[role]
			if !ok || rank < minRank {
				http.Error(w, `{"error":"insufficient permissions"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
