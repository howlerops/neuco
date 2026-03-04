// Package middleware provides reusable Chi middleware for the Neuco API.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/neuco-ai/neuco/internal/domain"
)

type contextKey int

const (
	ctxKeyUserID    contextKey = iota
	ctxKeyOrgID     contextKey = iota
	ctxKeyProjectID contextKey = iota
	ctxKeyRole      contextKey = iota
)

// NeuClaims are the custom JWT claims issued by the auth handler.
type NeuClaims struct {
	UserID string `json:"user_id"`
	OrgID  string `json:"org_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// Authenticate returns a middleware that validates a Bearer JWT from the
// Authorization header, parses the custom claims, and stores UserID, OrgID,
// Email, and Role into the request context. Requests without a valid token
// receive 401 Unauthorized.
func Authenticate(jwtSecret string) func(http.Handler) http.Handler {
	secret := []byte(jwtSecret)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				http.Error(w, `{"error":"malformed authorization header"}`, http.StatusUnauthorized)
				return
			}

			tokenStr := parts[1]
			claims := &NeuClaims{}

			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return secret, nil
			}, jwt.WithValidMethods([]string{"HS256"}))
			if err != nil || !token.Valid {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			userID, err := uuid.Parse(claims.UserID)
			if err != nil {
				http.Error(w, `{"error":"invalid user_id in token"}`, http.StatusUnauthorized)
				return
			}

			orgID, err := uuid.Parse(claims.OrgID)
			if err != nil {
				http.Error(w, `{"error":"invalid org_id in token"}`, http.StatusUnauthorized)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, ctxKeyUserID, userID)
			ctx = context.WithValue(ctx, ctxKeyOrgID, orgID)
			ctx = context.WithValue(ctx, ctxKeyRole, domain.OrgRole(claims.Role))

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromCtx retrieves the authenticated user's UUID from the context.
// Returns uuid.Nil if not set.
func UserIDFromCtx(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(ctxKeyUserID).(uuid.UUID)
	return v
}

// OrgIDFromCtx retrieves the current organisation UUID from the context.
// Returns uuid.Nil if not set.
func OrgIDFromCtx(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(ctxKeyOrgID).(uuid.UUID)
	return v
}

// RoleFromCtx retrieves the authenticated user's OrgRole from the context.
func RoleFromCtx(ctx context.Context) domain.OrgRole {
	v, _ := ctx.Value(ctxKeyRole).(domain.OrgRole)
	return v
}
