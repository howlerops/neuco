package domain

import (
	"time"

	"github.com/google/uuid"
)

// OnboardingStep identifies a single step in the onboarding flow.
type OnboardingStep string

const (
	OnboardingStepWelcome   OnboardingStep = "welcome"
	OnboardingStepOrg       OnboardingStep = "org"
	OnboardingStepProject   OnboardingStep = "project"
	OnboardingStepSignal    OnboardingStep = "signal"
	OnboardingStepSynthesis OnboardingStep = "synthesis"
	OnboardingStepDone      OnboardingStep = "done"
)

// AllOnboardingSteps lists the steps in order.
var AllOnboardingSteps = []OnboardingStep{
	OnboardingStepWelcome,
	OnboardingStepOrg,
	OnboardingStepProject,
	OnboardingStepSignal,
	OnboardingStepSynthesis,
	OnboardingStepDone,
}

// UserOnboarding tracks a user's onboarding progress.
type UserOnboarding struct {
	UserID         uuid.UUID        `json:"user_id"`
	CompletedSteps []OnboardingStep `json:"completed_steps"`
	CompletedAt    *time.Time       `json:"completed_at,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

// IsComplete returns true if the user finished onboarding.
func (u *UserOnboarding) IsComplete() bool {
	return u.CompletedAt != nil
}

// OnboardingStatus is returned by the onboarding API.
type OnboardingStatus struct {
	CompletedSteps []OnboardingStep `json:"completed_steps"`
	IsComplete     bool             `json:"is_complete"`
	TotalSteps     int              `json:"total_steps"`
}
