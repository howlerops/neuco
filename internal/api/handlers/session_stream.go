package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	mw "github.com/neuco-ai/neuco/internal/api/middleware"
	"github.com/neuco-ai/neuco/internal/store"
)

// sandboxSessionPage is the paginated list response for sandbox sessions.
type sandboxSessionPage struct {
	Sessions []store.SandboxSessionRow `json:"sessions"`
	Total    int                       `json:"total"`
}

// sandboxSessionStreamEvent is the SSE payload for sandbox session updates.
type sandboxSessionStreamEvent struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// GetSandboxSessionDetail handles GET /api/v1/projects/{projectId}/sessions/{sessionId}.
func GetSandboxSessionDetail(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())

		sessionID, err := uuid.Parse(chi.URLParam(r, "sessionId"))
		if err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid session_id")
			return
		}

		session, err := d.Store.GetSandboxSession(r.Context(), sessionID)
		if err != nil || session.ProjectID != projectID {
			respondErr(w, r, http.StatusNotFound, "sandbox session not found")
			return
		}

		respondOK(w, r, session)
	}
}

// ListSandboxSessions handles GET /api/v1/projects/{projectId}/sessions.
func ListSandboxSessions(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())

		limit := 20
		offset := 0
		if lStr := r.URL.Query().Get("limit"); lStr != "" {
			if n, err := strconv.Atoi(lStr); err == nil && n > 0 && n <= 200 {
				limit = n
			}
		}
		if oStr := r.URL.Query().Get("offset"); oStr != "" {
			if n, err := strconv.Atoi(oStr); err == nil && n >= 0 {
				offset = n
			}
		}

		sessions, total, err := d.Store.ListProjectSessions(r.Context(), projectID, store.Page(limit, offset))
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to list sandbox sessions")
			return
		}

		respondOK(w, r, sandboxSessionPage{Sessions: sessions, Total: total})
	}
}

// StopSandboxSession handles DELETE /api/v1/projects/{projectId}/sessions/{sessionId}.
func StopSandboxSession(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())

		sessionID, err := uuid.Parse(chi.URLParam(r, "sessionId"))
		if err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid session_id")
			return
		}

		session, err := d.Store.GetSandboxSession(r.Context(), sessionID)
		if err != nil || session.ProjectID != projectID {
			respondErr(w, r, http.StatusNotFound, "sandbox session not found")
			return
		}

		if isTerminalSessionStatus(session.Status) {
			respondNoContent(w, r)
			return
		}

		if err := d.Store.UpdateSandboxSessionStatus(r.Context(), sessionID, "cancelled", nil); err != nil {
			respondErr(w, r, http.StatusInternalServerError, "failed to stop sandbox session")
			return
		}

		respondNoContent(w, r)
	}
}

// StreamSandboxSession handles GET /api/v1/projects/{projectId}/sessions/{sessionId}/stream.
// Streams sandbox session state updates as Server-Sent Events, polling every 2s.
func StreamSandboxSession(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := mw.ProjectIDFromCtx(r.Context())

		sessionID, err := uuid.Parse(chi.URLParam(r, "sessionId"))
		if err != nil {
			respondErr(w, r, http.StatusBadRequest, "invalid session_id")
			return
		}

		session, err := d.Store.GetSandboxSession(r.Context(), sessionID)
		if err != nil || session.ProjectID != projectID {
			respondErr(w, r, http.StatusNotFound, "sandbox session not found")
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()

		writeSSE := func(eventType string, payload any) {
			envelope := sandboxSessionStreamEvent{Type: eventType, Data: payload}
			data, jErr := json.Marshal(envelope)
			if jErr != nil {
				data = []byte(`{"type":"error","data":{}}`)
			}
			_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(data))
			flusher.Flush()
		}

		writeSSE("status_update", session)
		if session.AgentLog != nil && strings.TrimSpace(*session.AgentLog) != "" {
			writeSSE("log_update", map[string]any{"agent_log": *session.AgentLog})
		}
		if len(session.ValidationResults) > 0 && string(session.ValidationResults) != "{}" {
			writeSSE("validation_result", map[string]json.RawMessage{"validation_results": session.ValidationResults})
		}
		if isTerminalSessionStatus(session.Status) {
			writeSSE("complete", map[string]string{"status": session.Status})
			return
		}

		lastStatus := session.Status
		lastLog := ""
		if session.AgentLog != nil {
			lastLog = *session.AgentLog
		}
		lastValidation := string(session.ValidationResults)

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				current, getErr := d.Store.GetSandboxSession(r.Context(), sessionID)
				if getErr != nil {
					writeSSE("complete", map[string]string{"status": "failed", "error": "failed to fetch sandbox session"})
					return
				}
				if current.ProjectID != projectID {
					writeSSE("complete", map[string]string{"status": "failed", "error": "sandbox session not found"})
					return
				}

				if current.Status != lastStatus {
					writeSSE("status_update", current)
					lastStatus = current.Status
				}

				currentLog := ""
				if current.AgentLog != nil {
					currentLog = *current.AgentLog
				}
				if currentLog != lastLog {
					writeSSE("log_update", map[string]any{"agent_log": currentLog})
					lastLog = currentLog
				}

				currentValidation := string(current.ValidationResults)
				if currentValidation != lastValidation {
					writeSSE("validation_result", map[string]json.RawMessage{"validation_results": current.ValidationResults})
					lastValidation = currentValidation
				}

				if isTerminalSessionStatus(current.Status) {
					writeSSE("complete", map[string]string{"status": current.Status})
					return
				}
			}
		}
	}
}

func isTerminalSessionStatus(status string) bool {
	switch status {
	case "completed", "failed", "cancelled", "timed_out":
		return true
	default:
		return false
	}
}
