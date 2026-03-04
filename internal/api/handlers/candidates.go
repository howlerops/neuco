package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/neuco-ai/neuco/internal/domain"
	"github.com/neuco-ai/neuco/internal/jobs"
	mw "github.com/neuco-ai/neuco/internal/api/middleware"
	"github.com/neuco-ai/neuco/internal/store"
	"github.com/riverqueue/river"
)

// updateCandidateStatusRequest is the request body for PATCH …/candidates/{cId}.
type updateCandidateStatusRequest struct {
	Status domain.CandidateStatus `json:"status"`
}

// refreshCandidatesResponse is returned when a synthesis job is enqueued.
type refreshCandidatesResponse struct {
	PipelineRunID string `json:"pipeline_run_id"`
}

// candidatePage is the paginated list response for candidates.
type candidatePage struct {
	Candidates []domain.FeatureCandidate `json:"candidates"`
	Total      int                       `json:"total"`
}

// ListCandidates handles GET /api/v1/projects/{projectId}/candidates.
func ListCandidates(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())

		limit := 50
		offset := 0
		if lStr := r.URL.Query().Get("limit"); lStr != "" {
			if n, err := strconv.Atoi(lStr); err == nil && n > 0 && n <= 500 {
				limit = n
			}
		}
		if oStr := r.URL.Query().Get("offset"); oStr != "" {
			if n, err := strconv.Atoi(oStr); err == nil && n >= 0 {
				offset = n
			}
		}

		candidates, total, err := d.Store.ListProjectCandidates(r.Context(), projectID, store.Page(limit, offset))
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to list candidates: "+err.Error())
			return
		}

		respondOK(w, r, candidatePage{Candidates: candidates, Total: total})
	}
}

// RefreshCandidates handles POST /api/v1/projects/{projectId}/candidates/refresh.
func RefreshCandidates(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())
		orgID := mw.OrgIDFromCtx(r.Context())

		// CreateSynthesisPipeline returns (runID, taskIDs, err).
		runID, taskIDs, err := jobs.CreateSynthesisPipeline(r.Context(), d.Store, projectID)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to create synthesis pipeline")
			return
		}

		// Enqueue the first synthesis task: fetch_signals.
		var firstTaskID uuid.UUID
		if len(taskIDs) > 0 {
			firstTaskID = taskIDs[0]
		}
		_, err = d.River.Insert(r.Context(), jobs.FetchSignalsJobArgs{
			ProjectID: projectID,
			RunID:     runID,
			TaskID:    firstTaskID,
		}, &river.InsertOpts{})
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to enqueue synthesis job")
			return
		}

		recordAudit(r.Context(), d, orgID, "candidate.refresh", "project", projectID.String(),
			map[string]any{"run_id": runID.String()})
		respondCreated(w, r, refreshCandidatesResponse{PipelineRunID: runID.String()})
	}
}

// UpdateCandidateStatus handles PATCH /api/v1/projects/{projectId}/candidates/{cId}.
func UpdateCandidateStatus(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())
		orgID := mw.OrgIDFromCtx(r.Context())

		candidateID, err := uuid.Parse(chi.URLParam(r, "cId"))
		if err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid candidate_id")
			return
		}

		var req updateCandidateStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Status == "" {
			respondErr(w, r, http.StatusBadRequest, "status is required")
			return
		}

		updated, err := d.Store.UpdateCandidateStatus(r.Context(), projectID, candidateID, req.Status)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to update candidate status")
			return
		}

		recordAudit(r.Context(), d, orgID, "candidate.status_change", "candidate", candidateID.String(),
			map[string]any{"new_status": req.Status})
		respondOK(w, r, updated)
	}
}
