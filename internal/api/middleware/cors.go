package middleware

import (
	"net/http"

	chiCors "github.com/go-chi/cors"
)

// CORS returns a chi-compatible CORS middleware that allows the frontend origin
// specified in frontendURL. All standard REST methods and common headers are
// permitted. Credentials (cookies, Authorization header) are allowed.
func CORS(frontendURL string) func(http.Handler) http.Handler {
	return chiCors.Handler(chiCors.Options{
		AllowedOrigins:   []string{frontendURL},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodHead},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Request-ID"},
		ExposedHeaders:   []string{"Link", "X-Total-Count"},
		AllowCredentials: true,
		MaxAge:           300,
	})
}
