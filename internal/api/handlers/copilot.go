package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/neuco-ai/neuco/internal/domain"
	mw "github.com/neuco-ai/neuco/internal/api/middleware"
)

// ListCopilotNotes handles GET /api/v1/projects/{projectId}/copilot/notes.
// Supports query params: target_type, target_id, include_dismissed.
func ListCopilotNotes(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())

		targetType := domain.CopilotNoteTargetType(r.URL.Query().Get("target_type"))
		var targetID *uuid.UUID
		if raw := r.URL.Query().Get("target_id"); raw != "" {
			if id, err := uuid.Parse(raw); err == nil {
				targetID = &id
			}
		}

		includeDismissed := r.URL.Query().Get("include_dismissed") == "true"

		notes, err := d.Store.ListCopilotNotes(r.Context(), projectID, targetType, targetID, includeDismissed)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to list copilot notes")
			return
		}

		respondOK(w, r, notes)
	}
}

// DismissCopilotNote handles PATCH /api/v1/projects/{projectId}/copilot/notes/{noteId}.
// Marks the note as dismissed.
func DismissCopilotNote(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())

		noteID, err := uuid.Parse(chi.URLParam(r, "noteId"))
		if err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid note_id")
			return
		}

		if err := d.Store.DismissCopilotNote(r.Context(), projectID, noteID); err != nil {
			respondErr(w, r, http.StatusNotFound, "note not found or already dismissed")
			return
		}

		respondOK(w, r, map[string]string{"status": "dismissed"})
	}
}
