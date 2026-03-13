package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	mw "github.com/neuco-ai/neuco/internal/api/middleware"
	"github.com/neuco-ai/neuco/internal/store"
)

// llmUsagePage is the paginated list response for LLM calls.
type llmUsagePage struct {
	Calls interface{} `json:"calls"`
	Total int         `json:"total"`
}

// GetProjectLLMUsage handles GET /api/v1/projects/{projectId}/llm-usage.
// Returns aggregated LLM usage stats for the project.
func GetProjectLLMUsage(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())

		agg, err := d.Store.GetLLMUsageByProject(r.Context(), projectID)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to get LLM usage")
			return
		}

		respondOK(w, r, agg)
	}
}

// ListProjectLLMCalls handles GET /api/v1/projects/{projectId}/llm-usage/calls.
// Returns paginated list of individual LLM calls.
func ListProjectLLMCalls(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())

		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		if limit <= 0 || limit > 100 {
			limit = 50
		}

		calls, total, err := d.Store.ListLLMCallsByProject(r.Context(), projectID, store.PageParams{Limit: limit, Offset: offset})
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to list LLM calls")
			return
		}

		respondOK(w, r, llmUsagePage{Calls: calls, Total: total})
	}
}

// GetOrgLLMUsage handles GET /api/v1/orgs/{orgId}/llm-usage.
// Returns aggregated LLM usage stats for all projects in the org.
func GetOrgLLMUsage(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := mw.ResolvedOrgIDFromCtx(r.Context())

		agg, err := d.Store.GetLLMUsageByOrg(r.Context(), orgID)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to get org LLM usage")
			return
		}

		respondOK(w, r, agg)
	}
}

// GetPipelineLLMUsage handles GET /api/v1/projects/{projectId}/pipelines/{runId}/llm-usage.
// Returns aggregated LLM usage for a specific pipeline run.
func GetPipelineLLMUsage(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		runIDStr := chi.URLParam(r, "runId")
		runID, err := uuid.Parse(runIDStr)
		if err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid run ID")
			return
		}

		agg, err := d.Store.GetLLMUsageByPipelineRun(r.Context(), runID)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to get pipeline LLM usage")
			return
		}

		respondOK(w, r, agg)
	}
}
