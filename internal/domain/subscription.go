package domain

import (
	"time"

	"github.com/google/uuid"
)

// PlanTier represents a billing tier.
type PlanTier string

const (
	PlanTierStarter PlanTier = "starter"
	PlanTierBuilder PlanTier = "builder"
)

// SubscriptionStatus tracks the Stripe subscription lifecycle.
type SubscriptionStatus string

const (
	SubStatusActive     SubscriptionStatus = "active"
	SubStatusPastDue    SubscriptionStatus = "past_due"
	SubStatusCanceled   SubscriptionStatus = "canceled"
	SubStatusIncomplete SubscriptionStatus = "incomplete"
	SubStatusTrialing   SubscriptionStatus = "trialing"
)

// Subscription holds Stripe billing data for an organisation.
type Subscription struct {
	ID                     uuid.UUID          `json:"id"`
	OrgID                  uuid.UUID          `json:"org_id"`
	StripeCustomerID       string             `json:"stripe_customer_id"`
	StripeSubscriptionID   *string            `json:"stripe_subscription_id,omitempty"`
	PlanTier               PlanTier           `json:"plan_tier"`
	Status                 SubscriptionStatus `json:"status"`
	CurrentPeriodEnd       *time.Time         `json:"current_period_end,omitempty"`
	CreatedAt              time.Time          `json:"created_at"`
	UpdatedAt              time.Time          `json:"updated_at"`
}

// PlanLimits returns the usage limits for a given tier.
type PlanLimits struct {
	MaxProjects  int `json:"max_projects"`
	MaxSignals   int `json:"max_signals_per_month"`
	MaxPRs       int `json:"max_prs_per_month"`
}

// FreeTierLimits are the limits applied when an org has no subscription.
var FreeTierLimits = PlanLimits{MaxProjects: 1, MaxSignals: 20, MaxPRs: 3}

// LimitsForTier returns the usage limits for a plan tier.
func LimitsForTier(tier PlanTier) PlanLimits {
	switch tier {
	case PlanTierBuilder:
		return PlanLimits{MaxProjects: 10, MaxSignals: 500, MaxPRs: 50}
	case PlanTierStarter:
		return PlanLimits{MaxProjects: 3, MaxSignals: 100, MaxPRs: 10}
	default:
		return FreeTierLimits
	}
}

// OrgUsage tracks per-period usage counters for an org.
type OrgUsage struct {
	ID           uuid.UUID `json:"id"`
	OrgID        uuid.UUID `json:"org_id"`
	PeriodStart  time.Time `json:"period_start"`
	SignalsCount int       `json:"signals_count"`
	PRsCount     int       `json:"prs_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UsageSummary is returned by the usage API.
type UsageSummary struct {
	Limits       PlanLimits `json:"limits"`
	ProjectCount int        `json:"project_count"`
	SignalsUsed  int        `json:"signals_used"`
	PRsUsed      int        `json:"prs_used"`
	PlanTier     *PlanTier  `json:"plan_tier"`
}
