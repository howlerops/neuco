package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/neuco-ai/neuco/internal/domain"
	"github.com/neuco-ai/neuco/internal/store"
)

// RequireActiveSubscription returns 402 if the org does not have an active
// subscription or valid free trial.
func RequireActiveSubscription(s *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orgID := OrgIDFromCtx(r.Context())

			sub, err := s.GetSubscriptionByOrgID(r.Context(), orgID)
			if err != nil {
				if err == pgx.ErrNoRows {
					// No subscription — allow free-tier access (limits
					// enforced separately by usage middleware).
					next.ServeHTTP(w, r)
					return
				}
				respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to check subscription"})
				return
			}

			if sub.Status != domain.SubStatusActive && sub.Status != domain.SubStatusTrialing {
				respondJSON(w, http.StatusPaymentRequired, map[string]any{
					"error": "subscription inactive",
					"code":  "inactive_subscription",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CheckSignalLimit returns 429 if the org has exhausted its signal upload limit.
func CheckSignalLimit(s *store.Store) func(http.Handler) http.Handler {
	return usageLimitMiddleware(s, func(limits domain.PlanLimits, usage domain.OrgUsage, _ int) (bool, string) {
		if usage.SignalsCount >= limits.MaxSignals {
			return true, fmt.Sprintf("signal limit reached (%d/%d)", usage.SignalsCount, limits.MaxSignals)
		}
		return false, ""
	})
}

// CheckProjectLimit returns 429 if the org has exhausted its project creation limit.
func CheckProjectLimit(s *store.Store) func(http.Handler) http.Handler {
	return usageLimitMiddleware(s, func(limits domain.PlanLimits, _ domain.OrgUsage, projectCount int) (bool, string) {
		if projectCount >= limits.MaxProjects {
			return true, fmt.Sprintf("project limit reached (%d/%d)", projectCount, limits.MaxProjects)
		}
		return false, ""
	})
}

// CheckPRLimit returns 429 if the org has exhausted its PR/codegen limit.
func CheckPRLimit(s *store.Store) func(http.Handler) http.Handler {
	return usageLimitMiddleware(s, func(limits domain.PlanLimits, usage domain.OrgUsage, _ int) (bool, string) {
		if usage.PRsCount >= limits.MaxPRs {
			return true, fmt.Sprintf("PR generation limit reached (%d/%d)", usage.PRsCount, limits.MaxPRs)
		}
		return false, ""
	})
}

// usageLimitMiddleware is the common logic: resolve the org's limits and
// current usage, then call the provided check function.
func usageLimitMiddleware(
	s *store.Store,
	check func(domain.PlanLimits, domain.OrgUsage, int) (exceeded bool, msg string),
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orgID := OrgIDFromCtx(r.Context())

			// Determine the plan limits.
			limits := domain.FreeTierLimits
			var tierName *domain.PlanTier
			sub, err := s.GetSubscriptionByOrgID(r.Context(), orgID)
			if err == nil && (sub.Status == domain.SubStatusActive || sub.Status == domain.SubStatusTrialing) {
				limits = domain.LimitsForTier(sub.PlanTier)
				tierName = &sub.PlanTier
			} else if err != nil && err != pgx.ErrNoRows {
				respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to check subscription"})
				return
			}

			// Check free-trial expiry (14 days from org creation) for orgs without a subscription.
			if tierName == nil {
				orgCreated, orgErr := s.GetOrgCreatedAt(r.Context(), orgID)
				if orgErr == nil && time.Since(orgCreated) > 14*24*time.Hour {
					respondJSON(w, http.StatusPaymentRequired, map[string]any{
						"error": "free trial expired",
						"code":  "trial_expired",
					})
					return
				}
			}

			usage, err := s.GetOrCreateUsage(r.Context(), orgID)
			if err != nil {
				respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to check usage"})
				return
			}

			projectCount, err := s.CountOrgProjects(r.Context(), orgID)
			if err != nil {
				respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to count projects"})
				return
			}

			if exceeded, msg := check(limits, usage, projectCount); exceeded {
				respondJSON(w, http.StatusTooManyRequests, map[string]any{
					"error":  msg,
					"code":   "usage_limit_exceeded",
					"limits": limits,
					"usage": map[string]int{
						"signals":  usage.SignalsCount,
						"prs":      usage.PRsCount,
						"projects": projectCount,
					},
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload) //nolint:errcheck
}
