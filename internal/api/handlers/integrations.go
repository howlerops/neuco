package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	mw "github.com/neuco-ai/neuco/internal/api/middleware"
	"github.com/neuco-ai/neuco/internal/domain"
)

type createIntegrationRequest struct {
	Provider string         `json:"provider"`
	Config   map[string]any `json:"config,omitempty"`
}

// ListIntegrations handles GET /api/v1/projects/{projectId}/integrations.
func ListIntegrations(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())

		intgs, err := d.Store.ListProjectIntegrations(r.Context(), projectID)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to list integrations")
			return
		}

		// Strip webhook secrets from list response
		for i := range intgs {
			intgs[i].WebhookSecret = ""
		}

		respondOK(w, r, intgs)
	}
}

// GetIntegration handles GET /api/v1/projects/{projectId}/integrations/{integrationId}.
func GetIntegration(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())
		integrationID, err := uuid.Parse(chi.URLParam(r, "integrationId"))
		if err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid integration_id")
			return
		}

		intg, err := d.Store.GetIntegration(r.Context(), projectID, integrationID)
		if err != nil {
			respondErr(w, r, http.StatusNotFound, "integration not found")
			return
		}

		intg.WebhookSecret = ""
		respondOK(w, r, intg)
	}
}

// CreateIntegration handles POST /api/v1/projects/{projectId}/integrations.
func CreateIntegration(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())

		var req createIntegrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Provider == "" {
			respondErr(w, r, http.StatusBadRequest, "provider is required")
			return
		}

		// Generate a 32-byte webhook secret
		secretBytes := make([]byte, 32)
		if _, err := rand.Read(secretBytes); err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to generate secret")
			return
		}
		secret := hex.EncodeToString(secretBytes)

		intg := domain.Integration{
			ProjectID:     projectID,
			Provider:      req.Provider,
			WebhookSecret: secret,
			Config:        req.Config,
			IsActive:      true,
		}

		created, err := d.Store.CreateIntegration(r.Context(), intg)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to create integration")
			return
		}

		// Return with webhook_secret visible (only time it's shown)
		respondCreated(w, r, created)
	}
}

// DeleteIntegration handles DELETE /api/v1/projects/{projectId}/integrations/{integrationId}.
func DeleteIntegration(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())
		integrationID, err := uuid.Parse(chi.URLParam(r, "integrationId"))
		if err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid integration_id")
			return
		}

		if err := d.Store.DeleteIntegration(r.Context(), projectID, integrationID); err != nil {
			respondErr(w, r, http.StatusNotFound, "integration not found")
			return
		}

		respondNoContent(w, r)
	}
}
