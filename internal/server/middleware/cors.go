package middleware

import (
	"net/http"
	"strings"
)

// CORS adds CORS headers for the configured origin(s); OFF by default
// (allowOrigin == "") because the daemon is internal-only.
//
// allowOrigin is "*" or a comma-separated list of exact origins; a matching
// Origin is echoed back, anything else gets no Access-Control-Allow-Origin.
// Vary: Origin is set whenever CORS is on, wildcard included, so a cache never
// serves one origin's response to another. Preflight OPTIONS answer 204.
func CORS(allowOrigin string) func(http.Handler) http.Handler {
	if allowOrigin == "" {
		return func(next http.Handler) http.Handler { return next }
	}
	wildcard := allowOrigin == "*"
	allowed := make(map[string]bool)
	if !wildcard {
		for _, o := range strings.Split(allowOrigin, ",") {
			if o = strings.TrimSpace(o); o != "" {
				allowed[o] = true
			}
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Add("Vary", "Origin")
			if origin := r.Header.Get("Origin"); wildcard {
				h.Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" && allowed[origin] {
				h.Set("Access-Control-Allow-Origin", origin)
			}
			h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			// The conditional reads and the gestures of a Direction (ADR-0057) speak in
			// these headers, and a reconnecting event stream sends Last-Event-ID; a browser
			// client must be allowed to send and to read them.
			h.Set("Access-Control-Allow-Headers", "Content-Type, "+TenantHeader+", If-Match, If-None-Match, Idempotency-Key, Last-Event-ID")
			h.Set("Access-Control-Expose-Headers", "ETag, Direction-Version, Idempotency-Replayed")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
