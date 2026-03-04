package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/neuco-ai/neuco/internal/domain"
	"github.com/neuco-ai/neuco/internal/jobs"
	"github.com/riverqueue/river"
)

// webhookPayload is the expected JSON body for the inbound webhook endpoint.
type webhookPayload struct {
	Content string          `json:"content"`
	Source  string          `json:"source,omitempty"`
	Type    string          `json:"type,omitempty"`
	Meta    json.RawMessage `json:"meta,omitempty"`
}

// webhookResponse is the response returned after successful webhook ingestion.
type webhookResponse struct {
	SignalID string `json:"signal_id"`
	Status   string `json:"status"`
}

// Webhook handles POST /api/v1/webhooks/{projectId}/{secret}.
// The URL embeds the integration's webhook secret so external callers can
// authenticate without a separate auth header. Validation uses the store
// method which performs a constant-time comparison internally.
func Webhook(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectIDStr := chi.URLParam(r, "projectId")
		secretParam := chi.URLParam(r, "secret")

		projectID, err := uuid.Parse(projectIDStr)
		if err != nil {
			http.Error(w, `{"error":"invalid project_id"}`, http.StatusBadRequest)
			return
		}

		// Find any integration for this project with a matching webhook secret.
		integrations, err := d.Store.ListProjectIntegrations(r.Context(), projectID)
		if err != nil {
			http.Error(w, `{"error":"invalid secret"}`, http.StatusUnauthorized)
			return
		}

		found := false
		for _, intg := range integrations {
			valid, verr := d.Store.ValidateWebhookSecret(r.Context(), projectID, intg.ID, secretParam)
			if verr == nil && valid {
				found = true
				break
			}
		}
		if !found {
			http.Error(w, `{"error":"invalid secret"}`, http.StatusUnauthorized)
			return
		}

		var payload webhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Content == "" {
			http.Error(w, `{"error":"content is required"}`, http.StatusBadRequest)
			return
		}

		src := domain.SignalSourceWebhook
		if payload.Source != "" {
			src = domain.SignalSource(payload.Source)
		}

		typ := domain.SignalTypeFeatureRequest
		if payload.Type != "" {
			typ = domain.SignalType(payload.Type)
		}

		meta := json.RawMessage("{}")
		if payload.Meta != nil {
			meta = payload.Meta
		}

		signalID := uuid.New()
		signal := domain.Signal{
			ID:         signalID,
			ProjectID:  projectID,
			Source:     src,
			Type:       typ,
			Content:    payload.Content,
			Metadata:   meta,
			OccurredAt: time.Now().UTC(),
		}

		if err := insertSignalWithJob(r.Context(), d, signal); err != nil {
			http.Error(w, `{"error":"failed to process webhook"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(webhookResponse{ //nolint:errcheck
			SignalID: signalID.String(),
			Status:   "accepted",
		})
	}
}

// insertSignalWithJob inserts the signal and enqueues an ingest job.
func insertSignalWithJob(ctx context.Context, d *Deps, signal domain.Signal) error {
	inserted, err := d.Store.InsertSignal(ctx, signal)
	if err != nil {
		return err
	}

	runID, taskIDs, err := jobs.CreateIngestPipeline(ctx, d.Store, signal.ProjectID)
	if err != nil {
		return err
	}

	payload, _ := json.Marshal(map[string]string{
		"content":    inserted.Content,
		"type":       string(inserted.Type),
		"source_ref": inserted.SourceRef,
	})

	_, err = d.River.Insert(ctx, jobs.IngestJobArgs{
		ProjectID:  signal.ProjectID,
		RawPayload: payload,
		Source:      string(signal.Source),
		RunID:      runID,
		TaskID:     taskIDs[0],
	}, &river.InsertOpts{Queue: "ingest"})
	return err
}
