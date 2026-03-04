package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/neuco-ai/neuco/internal/domain"
	"github.com/neuco-ai/neuco/internal/jobs"
	mw "github.com/neuco-ai/neuco/internal/api/middleware"
	"github.com/riverqueue/river"
)

// updateSpecRequest is the request body for PATCH …/spec (inline edit).
type updateSpecRequest struct {
	ProblemStatement   *string            `json:"problem_statement,omitempty"`
	ProposedSolution   *string            `json:"proposed_solution,omitempty"`
	UserStories        []domain.UserStory `json:"user_stories,omitempty"`
	AcceptanceCriteria []string           `json:"acceptance_criteria,omitempty"`
	OutOfScope         []string           `json:"out_of_scope,omitempty"`
	UIChanges          *string            `json:"ui_changes,omitempty"`
	DataModelChanges   *string            `json:"data_model_changes,omitempty"`
	OpenQuestions      []string           `json:"open_questions,omitempty"`
}

// generateSpecResponse is returned when a spec generation job is enqueued.
type generateSpecResponse struct {
	PipelineRunID string `json:"pipeline_run_id"`
}

// GetSpec handles GET /api/v1/projects/{projectId}/candidates/{cId}/spec.
func GetSpec(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())

		candidateID, err := uuid.Parse(chi.URLParam(r, "cId"))
		if err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid candidate_id")
			return
		}

		spec, err := d.Store.GetSpecByCandidate(r.Context(), projectID, candidateID)
		if err != nil {
			respondErr(w, r, http.StatusNotFound, "spec not found")
			return
		}

		respondOK(w, r, spec)
	}
}

// GenerateSpec handles POST /api/v1/projects/{projectId}/candidates/{cId}/spec/generate.
// Creates a pipeline run and enqueues a spec generation job.
func GenerateSpec(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())
		orgID := mw.OrgIDFromCtx(r.Context())

		candidateID, err := uuid.Parse(chi.URLParam(r, "cId"))
		if err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid candidate_id")
			return
		}

		if _, err := d.Store.GetCandidate(r.Context(), projectID, candidateID); err != nil {
			respondErr(w, r, http.StatusNotFound, "candidate not found")
			return
		}

		// CreateSpecGenPipeline creates the run and a single task, returning (runID, taskID, err).
		runID, taskID, err := jobs.CreateSpecGenPipeline(r.Context(), d.Store, projectID, candidateID)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to create pipeline")
			return
		}

		_, err = d.River.Insert(r.Context(), jobs.SpecGenJobArgs{
			CandidateID: candidateID,
			ProjectID:   projectID,
			RunID:       runID,
			TaskID:      taskID,
		}, &river.InsertOpts{})
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to enqueue spec generation job")
			return
		}

		recordAudit(r.Context(), d, orgID, "spec.generate", "spec", candidateID.String(),
			map[string]any{"run_id": runID.String()})
		respondCreated(w, r, generateSpecResponse{PipelineRunID: runID.String()})
	}
}

// UpdateSpec handles PATCH /api/v1/projects/{projectId}/candidates/{cId}/spec.
// Creates a new spec version via the store's UpdateSpec method (auto-bumps version).
func UpdateSpec(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())
		orgID := mw.OrgIDFromCtx(r.Context())
		userID := mw.UserIDFromCtx(r.Context())

		candidateID, err := uuid.Parse(chi.URLParam(r, "cId"))
		if err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid candidate_id")
			return
		}

		var req updateSpecRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid request body")
			return
		}

		existing, err := d.Store.GetSpecByCandidate(r.Context(), projectID, candidateID)
		if err != nil {
			respondErr(w, r, http.StatusNotFound, "spec not found — generate one first")
			return
		}

		patch := existing
		if req.ProblemStatement != nil {
			patch.ProblemStatement = *req.ProblemStatement
		}
		if req.ProposedSolution != nil {
			patch.ProposedSolution = *req.ProposedSolution
		}
		if req.UserStories != nil {
			patch.UserStories = req.UserStories
		}
		if req.AcceptanceCriteria != nil {
			patch.AcceptanceCriteria = req.AcceptanceCriteria
		}
		if req.OutOfScope != nil {
			patch.OutOfScope = req.OutOfScope
		}
		if req.UIChanges != nil {
			patch.UIChanges = *req.UIChanges
		}
		if req.DataModelChanges != nil {
			patch.DataModelChanges = *req.DataModelChanges
		}
		if req.OpenQuestions != nil {
			patch.OpenQuestions = req.OpenQuestions
		}
		patch.GeneratedBy = &userID

		// UpdateSpec(ctx, projectID, candidateID, patch) auto-bumps the version in a tx.
		updated, err := d.Store.UpdateSpec(r.Context(), projectID, candidateID, patch)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to update spec")
			return
		}

		recordAudit(r.Context(), d, orgID, "spec.update", "spec", existing.ID.String(),
			map[string]any{"new_version": updated.Version})
		respondOK(w, r, updated)
	}
}
