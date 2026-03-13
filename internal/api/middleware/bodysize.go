package middleware

import "net/http"

// MaxBodySize returns middleware that limits the request body to maxBytes.
// If the body exceeds the limit, http.MaxBytesReader causes the next Read
// to return an error, which json.Decoder surfaces as a 400.
func MaxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}
