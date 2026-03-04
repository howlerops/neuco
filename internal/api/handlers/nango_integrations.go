package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/riverqueue/river"

	mw "github.com/neuco-ai/neuco/internal/api/middleware"
	"github.com/neuco-ai/neuco/internal/domain"
	"github.com/neuco-ai/neuco/internal/jobs"
	"github.com/neuco-ai/neuco/internal/nango"
)

// nangoConnectionResponse is the view-model returned to the frontend.
type nangoConnectionResponse struct {
	IntegrationID     uuid.UUID  `json:"integration_id"`
	Provider          string     `json:"provider"`
	ProviderConfigKey string     `json:"provider_config_key"`
	ConnectionID      string     `json:"connection_id"`
	IsActive          bool       `json:"is_active"`
	LastSyncAt        *string    `json:"last_sync_at,omitempty"`
	CreatedAt         string     `json:"created_at"`
}

// createNangoConnectionRequest is the body for POST .../nango/connections.
// The frontend sends this after the Nango frontend SDK completes the OAuth
// flow and yields a connectionId.
type createNangoConnectionRequest struct {
	ProviderConfigKey string `json:"provider_config_key"`
	ConnectionID      string `json:"connection_id"`
}

// triggerSyncResponse is the body returned after a manual sync is enqueued.
type triggerSyncResponse struct {
	IntegrationID string `json:"integration_id"`
	ConnectionID  string `json:"connection_id"`
	Message       string `json:"message"`
}

// nangoClient builds a Nango client from handler deps.
func nangoClient(d *Deps) *nango.Client {
	return nango.NewClient(d.Config.NangoServerURL, d.Config.NangoSecretKey)
}

// CreateNangoConnectSession handles POST /api/v1/auth/nango/connect-session.
//
// Creates a short-lived Nango Connect session token for the authenticated user.
// The frontend uses this token to initialise the Nango frontend SDK for OAuth
// flows, replacing the deprecated public-key approach.
func CreateNangoConnectSession(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := mw.UserIDFromCtx(r.Context())

		user, err := d.Store.GetUserByID(r.Context(), userID)
		if err != nil {
			slog.ErrorContext(r.Context(), "nango: failed to fetch user for connect session",
				"user_id", userID,
				"error", err,
			)
			respondErr(w, r, http.StatusInternalServerError, "failed to retrieve user")
			return
		}

		nc := nangoClient(d)

		token, err := nc.CreateConnectSession(r.Context(),
			user.ID.String(),
			user.Email,
			user.GitHubLogin,
		)
		if err != nil {
			slog.ErrorContext(r.Context(), "nango: failed to create connect session",
				"user_id", userID,
				"error", err,
			)
			respondErr(w, r, http.StatusBadGateway, "failed to create nango connect session")
			return
		}

		respondOK(w, r, map[string]string{"token": token})
	}
}

// ListNangoConnections handles GET /api/v1/projects/{projectId}/nango/connections.
//
// Returns all integration records for this project that were created via the
// Nango OAuth flow (identified by a non-empty connection_id in the config).
func ListNangoConnections(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())

		intgs, err := d.Store.ListProjectIntegrations(r.Context(), projectID)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to list integrations")
			return
		}

		var out []nangoConnectionResponse
		for _, intg := range intgs {
			connID, _ := intg.Config["connection_id"].(string)
			pcKey, _ := intg.Config["provider_config_key"].(string)
			if connID == "" {
				// Not a Nango-managed integration; skip.
				continue
			}

			var lastSync *string
			if intg.LastSyncAt != nil {
				s := intg.LastSyncAt.UTC().Format("2006-01-02T15:04:05Z")
				lastSync = &s
			}

			out = append(out, nangoConnectionResponse{
				IntegrationID:     intg.ID,
				Provider:          intg.Provider,
				ProviderConfigKey: pcKey,
				ConnectionID:      connID,
				IsActive:          intg.IsActive,
				LastSyncAt:        lastSync,
				CreatedAt:         intg.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			})
		}

		if out == nil {
			out = []nangoConnectionResponse{}
		}
		respondOK(w, r, out)
	}
}

// CreateNangoConnection handles POST /api/v1/projects/{projectId}/nango/connections.
//
// Called by the frontend after the Nango frontend SDK has completed the OAuth
// flow. This handler verifies the connection exists in Nango, then persists an
// integration record so we can reference it during syncs.
func CreateNangoConnection(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())
		orgID := mw.OrgIDFromCtx(r.Context())

		var req createNangoConnectionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.ProviderConfigKey == "" || req.ConnectionID == "" {
			respondErr(w, r, http.StatusBadRequest, "provider_config_key and connection_id are required")
			return
		}

		nc := nangoClient(d)

		// Verify the connection actually exists in Nango before storing it.
		conn, err := nc.GetConnection(r.Context(), req.ProviderConfigKey, req.ConnectionID)
		if err != nil {
			slog.ErrorContext(r.Context(), "nango: connection not found",
				"provider_config_key", req.ProviderConfigKey,
				"connection_id", req.ConnectionID,
				"error", err,
			)
			respondErr(w, r, http.StatusBadRequest, "nango connection not found or inaccessible")
			return
		}

		intg := domain.Integration{
			ProjectID: projectID,
			Provider:  conn.Provider,
			Config: map[string]any{
				"provider_config_key": req.ProviderConfigKey,
				"connection_id":       req.ConnectionID,
				"nango_id":            conn.ID,
			},
			IsActive: true,
		}

		created, err := d.Store.CreateIntegration(r.Context(), intg)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to create integration")
			return
		}

		recordAudit(r.Context(), d, orgID, "nango_connection.create", "integration", created.ID.String(),
			map[string]any{
				"provider":            conn.Provider,
				"provider_config_key": req.ProviderConfigKey,
				"connection_id":       req.ConnectionID,
			})

		respondCreated(w, r, nangoConnectionResponse{
			IntegrationID:     created.ID,
			Provider:          created.Provider,
			ProviderConfigKey: req.ProviderConfigKey,
			ConnectionID:      req.ConnectionID,
			IsActive:          created.IsActive,
			CreatedAt:         created.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
}

// DeleteNangoConnection handles DELETE /api/v1/projects/{projectId}/nango/connections/{connectionId}.
//
// connectionId here is the UUID of the integration record in our DB. The
// handler also tells Nango to delete the connection on its side so the OAuth
// token is revoked.
func DeleteNangoConnection(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())
		orgID := mw.OrgIDFromCtx(r.Context())

		integrationID, err := uuid.Parse(chi.URLParam(r, "connectionId"))
		if err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid connection_id")
			return
		}

		intg, err := d.Store.GetIntegration(r.Context(), projectID, integrationID)
		if err != nil {
			respondErr(w, r, http.StatusNotFound, "integration not found")
			return
		}

		nangoConnID, _ := intg.Config["connection_id"].(string)
		nangoProviderKey, _ := intg.Config["provider_config_key"].(string)

		// Best-effort: tell Nango to revoke the connection. Log but do not
		// abort if Nango is unreachable — the local record should still be
		// removed so the user is not stuck.
		if nangoConnID != "" && nangoProviderKey != "" {
			nc := nangoClient(d)
			if delErr := nc.DeleteConnection(r.Context(), nangoProviderKey, nangoConnID); delErr != nil {
				slog.WarnContext(r.Context(), "nango: failed to delete connection from Nango (continuing with local delete)",
					"integration_id", integrationID,
					"nango_connection_id", nangoConnID,
					"error", delErr,
				)
			}
		}

		if err := d.Store.DeleteIntegration(r.Context(), projectID, integrationID); err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to delete integration")
			return
		}

		recordAudit(r.Context(), d, orgID, "nango_connection.delete", "integration", integrationID.String(),
			map[string]any{
				"provider":            intg.Provider,
				"provider_config_key": nangoProviderKey,
				"connection_id":       nangoConnID,
			})

		respondNoContent(w, r)
	}
}

// TriggerNangoSync handles POST /api/v1/projects/{projectId}/nango/sync/{connectionId}.
//
// Enqueues a NangoSyncJob for the given integration so data is fetched in the
// background. The connectionId URL param is the integration record UUID.
func TriggerNangoSync(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())

		integrationID, err := uuid.Parse(chi.URLParam(r, "connectionId"))
		if err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid connection_id")
			return
		}

		intg, err := d.Store.GetIntegration(r.Context(), projectID, integrationID)
		if err != nil {
			respondErr(w, r, http.StatusNotFound, "integration not found")
			return
		}

		nangoConnID, _ := intg.Config["connection_id"].(string)
		if nangoConnID == "" {
			respondErr(w, r, http.StatusBadRequest, "integration has no nango connection_id")
			return
		}

		// Create a minimal pipeline run so the sync can report task progress.
		runID, taskIDs, err := jobs.CreateNangoSyncPipeline(r.Context(), d.Store, projectID)
		if err != nil {
			slog.ErrorContext(r.Context(), "nango: failed to create sync pipeline",
				"integration_id", integrationID,
				"error", err,
			)
			// Non-fatal: enqueue the job without pipeline tracking.
			runID = uuid.Nil
		}

		var syncTaskID uuid.UUID
		if len(taskIDs) > 0 {
			syncTaskID = taskIDs[0]
		}

		args := jobs.NangoSyncJobArgs{
			ProjectID:    projectID,
			ConnectionID: nangoConnID,
			Provider:     intg.Provider,
			RunID:        runID,
			TaskID:       syncTaskID,
			IntegrationID: intg.ID,
		}

		if _, err := d.River.Insert(r.Context(), args, &river.InsertOpts{
			Queue: "ingest",
		}); err != nil {
			respondErr(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to enqueue sync job: %s", err))
			return
		}

		respondOK(w, r, triggerSyncResponse{
			IntegrationID: integrationID.String(),
			ConnectionID:  nangoConnID,
			Message:       "sync job enqueued",
		})
	}
}
