package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/neuco-ai/neuco/internal/api/middleware"
	"github.com/neuco-ai/neuco/internal/domain"
	"github.com/stripe/stripe-go/v82"
	billingPortalSession "github.com/stripe/stripe-go/v82/billingportal/session"
	checkoutSession "github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/webhook"
)

// CreateCheckoutSession creates a Stripe Checkout session for a given plan tier.
func CreateCheckoutSession(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := middleware.OrgIDFromCtx(r.Context())
		userID := middleware.UserIDFromCtx(r.Context())

		var req struct {
			PlanTier string `json:"plan_tier"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid request body")
			return
		}

		var priceID string
		switch domain.PlanTier(req.PlanTier) {
		case domain.PlanTierStarter:
			priceID = d.Config.StripePriceStarter
		case domain.PlanTierBuilder:
			priceID = d.Config.StripePriceBuilder
		default:
			respondErr(w, r, http.StatusBadRequest, "invalid plan_tier: must be 'starter' or 'builder'")
			return
		}
		if priceID == "" {
			respondErr(w, r, http.StatusInternalServerError, "stripe price not configured")
			return
		}

		stripe.Key = d.Config.StripeSecretKey

		params := &stripe.CheckoutSessionParams{
			Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
			LineItems: []*stripe.CheckoutSessionLineItemParams{
				{Price: stripe.String(priceID), Quantity: stripe.Int64(1)},
			},
			SuccessURL: stripe.String(d.Config.FrontendURL + "/billing?status=success"),
			CancelURL:  stripe.String(d.Config.FrontendURL + "/billing?status=canceled"),
			ClientReferenceID: stripe.String(orgID.String()),
			Metadata: map[string]string{
				"org_id":  orgID.String(),
				"user_id": userID.String(),
				"plan":    req.PlanTier,
			},
		}

		// If org already has a Stripe customer, use it.
		sub, err := d.Store.GetSubscriptionByOrgID(r.Context(), orgID)
		if err == nil {
			params.Customer = stripe.String(sub.StripeCustomerID)
		}

		sess, err := checkoutSession.New(params)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to create checkout session")
			return
		}

		respondOK(w, r, map[string]string{"url": sess.URL})
	}
}

// CreatePortalSession creates a Stripe Customer Portal session for self-service billing.
func CreatePortalSession(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := middleware.OrgIDFromCtx(r.Context())

		sub, err := d.Store.GetSubscriptionByOrgID(r.Context(), orgID)
		if err != nil {
			respondErr(w, r, http.StatusNotFound, "no subscription found")
			return
		}

		stripe.Key = d.Config.StripeSecretKey
		params := &stripe.BillingPortalSessionParams{
			Customer:  stripe.String(sub.StripeCustomerID),
			ReturnURL: stripe.String(d.Config.FrontendURL + "/billing"),
		}

		sess, err := billingPortalSession.New(params)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to create portal session")
			return
		}

		respondOK(w, r, map[string]string{"url": sess.URL})
	}
}

// GetSubscription returns the current subscription for an org.
func GetSubscription(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := middleware.OrgIDFromCtx(r.Context())

		sub, err := d.Store.GetSubscriptionByOrgID(r.Context(), orgID)
		if err != nil {
			if err == pgx.ErrNoRows {
				respondOK(w, r, map[string]any{"subscription": nil})
				return
			}
			respondErr(w, r, http.StatusInternalServerError, "failed to get subscription")
			return
		}

		limits := domain.LimitsForTier(sub.PlanTier)
		respondOK(w, r, map[string]any{
			"subscription": sub,
			"limits":       limits,
		})
	}
}

// GetUsage returns current usage vs limits for the org.
func GetUsage(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := middleware.OrgIDFromCtx(r.Context())

		// Determine plan limits.
		limits := domain.FreeTierLimits
		var tierName *domain.PlanTier
		sub, err := d.Store.GetSubscriptionByOrgID(r.Context(), orgID)
		if err == nil {
			limits = domain.LimitsForTier(sub.PlanTier)
			tierName = &sub.PlanTier
		}

		usage, err := d.Store.GetOrCreateUsage(r.Context(), orgID)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to get usage")
			return
		}

		projectCount, err := d.Store.CountOrgProjects(r.Context(), orgID)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to count projects")
			return
		}

		respondOK(w, r, domain.UsageSummary{
			Limits:       limits,
			ProjectCount: projectCount,
			SignalsUsed:  usage.SignalsCount,
			PRsUsed:      usage.PRsCount,
			PlanTier:     tierName,
		})
	}
}

// StripeWebhook handles Stripe webhook events.
func StripeWebhook(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 65536))
		if err != nil {
			respondErr(w, r, http.StatusBadRequest, "failed to read body")
			return
		}

		event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), d.Config.StripeWebhookSecret)
		if err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid webhook signature")
			return
		}

		switch event.Type {
		case "checkout.session.completed":
			handleCheckoutCompleted(d, r, event)
		case "customer.subscription.updated":
			handleSubscriptionUpdated(d, r, event)
		case "customer.subscription.deleted":
			handleSubscriptionDeleted(d, r, event)
		default:
			slog.DebugContext(r.Context(), "unhandled stripe event", "type", event.Type)
		}

		w.WriteHeader(http.StatusOK)
	}
}

func handleCheckoutCompleted(d *Deps, r *http.Request, event stripe.Event) {
	var sess stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
		slog.ErrorContext(r.Context(), "failed to parse checkout session", "error", err)
		return
	}

	orgID := sess.Metadata["org_id"]
	planTier := domain.PlanTier(sess.Metadata["plan"])

	if orgID == "" || sess.Customer == nil {
		slog.ErrorContext(r.Context(), "checkout session missing org_id or customer")
		return
	}

	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		slog.ErrorContext(r.Context(), "invalid org_id in checkout metadata", "org_id", orgID)
		return
	}

	var stripeSubID *string
	if sess.Subscription != nil {
		stripeSubID = &sess.Subscription.ID
	}

	sub := domain.Subscription{
		OrgID:                orgUUID,
		StripeCustomerID:     sess.Customer.ID,
		StripeSubscriptionID: stripeSubID,
		PlanTier:             planTier,
		Status:               domain.SubStatusActive,
	}

	if _, err := d.Store.UpsertSubscription(r.Context(), sub); err != nil {
		slog.ErrorContext(r.Context(), "failed to upsert subscription", "error", err, "org_id", orgID)
	}
}

func handleSubscriptionUpdated(d *Deps, r *http.Request, event stripe.Event) {
	var stripeSub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &stripeSub); err != nil {
		slog.ErrorContext(r.Context(), "failed to parse subscription update", "error", err)
		return
	}

	status := mapStripeStatus(stripeSub.Status)

	// Look up by customer ID and update.
	sub, err := d.Store.GetSubscriptionByStripeCustomerID(r.Context(), stripeSub.Customer.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "subscription not found for customer", "customer_id", stripeSub.Customer.ID)
		return
	}

	sub.Status = status
	sub.StripeSubscriptionID = &stripeSub.ID

	if _, err := d.Store.UpsertSubscription(r.Context(), sub); err != nil {
		slog.ErrorContext(r.Context(), "failed to update subscription", "error", err)
	}
}

func handleSubscriptionDeleted(d *Deps, r *http.Request, event stripe.Event) {
	var stripeSub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &stripeSub); err != nil {
		slog.ErrorContext(r.Context(), "failed to parse subscription deletion", "error", err)
		return
	}

	sub, err := d.Store.GetSubscriptionByStripeCustomerID(r.Context(), stripeSub.Customer.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "subscription not found for deletion", "customer_id", stripeSub.Customer.ID)
		return
	}

	sub.Status = domain.SubStatusCanceled
	if _, err := d.Store.UpsertSubscription(r.Context(), sub); err != nil {
		slog.ErrorContext(r.Context(), "failed to cancel subscription", "error", err)
	}
}

func mapStripeStatus(s stripe.SubscriptionStatus) domain.SubscriptionStatus {
	switch s {
	case stripe.SubscriptionStatusActive:
		return domain.SubStatusActive
	case stripe.SubscriptionStatusPastDue:
		return domain.SubStatusPastDue
	case stripe.SubscriptionStatusCanceled:
		return domain.SubStatusCanceled
	case stripe.SubscriptionStatusTrialing:
		return domain.SubStatusTrialing
	default:
		return domain.SubStatusIncomplete
	}
}

