package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/neuco-ai/neuco/internal/api/middleware"
	"github.com/neuco-ai/neuco/internal/domain"
)

// GetOnboardingStatus returns the current user's onboarding progress.
func GetOnboardingStatus(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.UserIDFromCtx(r.Context())

		ob, err := d.Store.GetOnboarding(r.Context(), userID)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to get onboarding status")
			return
		}

		respondOK(w, r, domain.OnboardingStatus{
			CompletedSteps: ob.CompletedSteps,
			IsComplete:     ob.IsComplete(),
			TotalSteps:     len(domain.AllOnboardingSteps),
		})
	}
}

// CompleteOnboardingStep marks a single step as done.
func CompleteOnboardingStep(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.UserIDFromCtx(r.Context())

		var req struct {
			Step string `json:"step"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid request body")
			return
		}

		step := domain.OnboardingStep(req.Step)
		valid := false
		for _, s := range domain.AllOnboardingSteps {
			if s == step {
				valid = true
				break
			}
		}
		if !valid {
			respondErr(w, r, http.StatusBadRequest, "invalid onboarding step")
			return
		}

		ob, err := d.Store.CompleteOnboardingStep(r.Context(), userID, step)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to complete step")
			return
		}

		respondOK(w, r, domain.OnboardingStatus{
			CompletedSteps: ob.CompletedSteps,
			IsComplete:     ob.IsComplete(),
			TotalSteps:     len(domain.AllOnboardingSteps),
		})
	}
}

// SkipOnboarding marks the entire onboarding as complete (skipped).
func SkipOnboarding(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.UserIDFromCtx(r.Context())

		ob, err := d.Store.CompleteOnboarding(r.Context(), userID)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to skip onboarding")
			return
		}

		respondOK(w, r, domain.OnboardingStatus{
			CompletedSteps: ob.CompletedSteps,
			IsComplete:     ob.IsComplete(),
			TotalSteps:     len(domain.AllOnboardingSteps),
		})
	}
}
