package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	mw "github.com/neuco-ai/neuco/internal/api/middleware"
	"github.com/neuco-ai/neuco/internal/domain"
	"github.com/neuco-ai/neuco/internal/store"
)

type createContextRequest struct {
	Category string `json:"category"`
	Title    string `json:"title"`
	Content  string `json:"content"`
}

type updateContextRequest struct {
	Category string `json:"category"`
	Title    string `json:"title"`
	Content  string `json:"content"`
}

type contextPage struct {
	Contexts []domain.ProjectContext `json:"contexts"`
	Total    int                     `json:"total"`
}

// ListProjectContexts handles GET /api/v1/projects/{projectId}/contexts.
func ListProjectContexts(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())

		limit := 50
		offset := 0
		if lStr := r.URL.Query().Get("limit"); lStr != "" {
			if n, err := strconv.Atoi(lStr); err == nil && n > 0 {
				limit = n
			}
		}
		limit = clampPagination(limit)
		if oStr := r.URL.Query().Get("offset"); oStr != "" {
			if n, err := strconv.Atoi(oStr); err == nil && n >= 0 {
				offset = n
			}
		}
		category := r.URL.Query().Get("category")

		contexts, total, err := d.Store.ListProjectContexts(r.Context(), projectID, category, store.Page(limit, offset))
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to list contexts")
			return
		}

		if contexts == nil {
			contexts = []domain.ProjectContext{}
		}

		respondOK(w, r, contextPage{Contexts: contexts, Total: total})
	}
}

// GetProjectContext handles GET /api/v1/projects/{projectId}/contexts/{contextId}.
func GetProjectContext(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())
		contextID, err := uuid.Parse(chi.URLParam(r, "contextId"))
		if err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid context ID")
			return
		}

		pc, err := d.Store.GetProjectContext(r.Context(), projectID, contextID)
		if err != nil {
			respondErr(w, r, http.StatusNotFound, "context not found")
			return
		}

		respondOK(w, r, pc)
	}
}

// CreateProjectContext handles POST /api/v1/projects/{projectId}/contexts.
func CreateProjectContext(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())
		orgID := mw.OrgIDFromCtx(r.Context())

		var req createContextRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Title == "" || req.Content == "" {
			respondErr(w, r, http.StatusBadRequest, "title and content are required")
			return
		}
		if msg := validateStringLen("title", req.Title, MaxTitleLen); msg != "" {
			respondErr(w, r, http.StatusBadRequest, msg)
			return
		}
		if msg := validateStringLen("content", req.Content, MaxContentLen); msg != "" {
			respondErr(w, r, http.StatusBadRequest, msg)
			return
		}

		category := domain.ContextCategory(req.Category)
		if category == "" {
			category = domain.ContextCategoryInsight
		}
		if !isValidContextCategory(category) {
			respondErr(w, r, http.StatusBadRequest, "invalid category: must be insight, theme, decision, risk, or opportunity")
			return
		}

		pc := domain.ProjectContext{
			ProjectID: projectID,
			Category:  category,
			Title:     req.Title,
			Content:   req.Content,
		}

		inserted, err := d.Store.InsertProjectContext(r.Context(), pc)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to create context")
			return
		}

		recordAudit(r.Context(), d, orgID, "context.create", "project_context", inserted.ID.String(),
			map[string]any{"project_id": projectID.String(), "title": inserted.Title})

		respondCreated(w, r, inserted)
	}
}

// UpdateProjectContext handles PATCH /api/v1/projects/{projectId}/contexts/{contextId}.
func UpdateProjectContext(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())
		orgID := mw.OrgIDFromCtx(r.Context())

		contextID, err := uuid.Parse(chi.URLParam(r, "contextId"))
		if err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid context ID")
			return
		}

		var req updateContextRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Title == "" || req.Content == "" {
			respondErr(w, r, http.StatusBadRequest, "title and content are required")
			return
		}
		if msg := validateStringLen("title", req.Title, MaxTitleLen); msg != "" {
			respondErr(w, r, http.StatusBadRequest, msg)
			return
		}
		if msg := validateStringLen("content", req.Content, MaxContentLen); msg != "" {
			respondErr(w, r, http.StatusBadRequest, msg)
			return
		}

		category := req.Category
		if category == "" {
			category = string(domain.ContextCategoryInsight)
		}
		if !isValidContextCategory(domain.ContextCategory(category)) {
			respondErr(w, r, http.StatusBadRequest, "invalid category: must be insight, theme, decision, risk, or opportunity")
			return
		}

		updated, err := d.Store.UpdateProjectContext(r.Context(), projectID, contextID, req.Title, req.Content, category)
		if err != nil {
			respondErr(w, r, http.StatusNotFound, "context not found")
			return
		}

		recordAudit(r.Context(), d, orgID, "context.update", "project_context", contextID.String(),
			map[string]any{"project_id": projectID.String()})

		respondOK(w, r, updated)
	}
}

// DeleteProjectContext handles DELETE /api/v1/projects/{projectId}/contexts/{contextId}.
func DeleteProjectContext(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())
		orgID := mw.OrgIDFromCtx(r.Context())

		contextID, err := uuid.Parse(chi.URLParam(r, "contextId"))
		if err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid context ID")
			return
		}

		if err := d.Store.DeleteProjectContext(r.Context(), projectID, contextID); err != nil {
			respondErr(w, r, http.StatusNotFound, "context not found")
			return
		}

		recordAudit(r.Context(), d, orgID, "context.delete", "project_context", contextID.String(),
			map[string]any{"project_id": projectID.String()})

		respondNoContent(w, r)
	}
}

// SearchProjectContexts handles POST /api/v1/projects/{projectId}/contexts/search.
func SearchProjectContexts(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())

		var req struct {
			Query string `json:"query"`
			Limit int    `json:"limit"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Query == "" {
			respondErr(w, r, http.StatusBadRequest, "query is required")
			return
		}

		if req.Limit <= 0 || req.Limit > 50 {
			req.Limit = 10
		}

		embedding, err := d.QueryEngine.GenerateEmbedding(r.Context(), req.Query)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to generate embedding")
			return
		}

		results, err := d.Store.SearchProjectContexts(r.Context(), projectID, embedding, req.Limit)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to search contexts")
			return
		}

		if results == nil {
			results = []store.ContextSearchResult{}
		}

		respondOK(w, r, map[string]any{
			"results": results,
			"total":   len(results),
		})
	}
}
