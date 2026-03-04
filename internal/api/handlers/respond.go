package handlers

import (
	"net/http"

	"github.com/go-chi/render"
)

// errResponse is the canonical JSON error body returned by all handlers.
type errResponse struct {
	HTTPStatusCode int    `json:"-"`
	Error          string `json:"error"`
}

func (e *errResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}

func respondErr(w http.ResponseWriter, r *http.Request, status int, msg string) {
	render.Render(w, r, &errResponse{HTTPStatusCode: status, Error: msg}) //nolint:errcheck
}

func respondOK(w http.ResponseWriter, r *http.Request, payload any) {
	render.JSON(w, r, payload)
}

func respondCreated(w http.ResponseWriter, r *http.Request, payload any) {
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, payload)
}

func respondNoContent(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
