package middleware

import "net/http"

// routeLabel returns the path for a known route, else "unmatched", keeping
// metric cardinality bounded against probes. The RPC API has no path
// parameters, so a known path is itself bounded.
func routeLabel(r *http.Request, known map[string]bool) string {
	if known[r.URL.Path] {
		return r.URL.Path
	}
	return "unmatched"
}
