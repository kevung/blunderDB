package middleware

import "net/http"

// CORS adds permissive CORS headers for the configured origin. It is OFF by
// default (allowOrigin == "") because the daemon is internal-only; a single
// origin (or "*") can be enabled for non-production setups via the serve flag.
//
// When enabled, preflight OPTIONS requests are answered with 204 directly.
func CORS(allowOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if allowOrigin == "" {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", allowOrigin)
			h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Content-Type, "+TenantHeader)
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
