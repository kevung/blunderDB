package middleware

import (
	"net/http"
	"strings"
)

// CORS adds permissive CORS headers for the configured origin(s). It is OFF
// by default (allowOrigin == "") because the daemon is internal-only.
//
// allowOrigin is either "*" or a comma-separated list of exact origins
// (e.g. "https://a.example, https://b.example"): a request whose Origin
// header matches one of them gets that same origin echoed back as
// Access-Control-Allow-Origin, so more than one legitimate front-end can
// share a daemon without every one of them being handed "*". A request whose
// Origin matches none of them (or a same-origin request with no Origin
// header at all) gets no Access-Control-Allow-Origin, same as CORS disabled.
//
// Vary: Origin is set on every response once CORS is enabled at all,
// wildcard included: which origin (if any) is echoed back depends on the
// request's Origin header, so a cache sitting in front of this daemon must
// not serve the response computed for one origin to a different one's
// request (#232).
//
// When enabled, preflight OPTIONS requests are answered with 204 directly.
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
			h.Set("Access-Control-Allow-Headers", "Content-Type, "+TenantHeader)
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
