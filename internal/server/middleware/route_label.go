package middleware

import "net/http"

// routeLabel returns a bounded-cardinality label for a request: the request
// path when it is a known route, otherwise the constant "unmatched". This
// keeps metric/label cardinality bounded even when clients probe arbitrary
// 404 paths. The RPC API has no path parameters, so a known path is itself a
// safe, bounded label.
func routeLabel(r *http.Request, known map[string]bool) string {
	if known[r.URL.Path] {
		return r.URL.Path
	}
	return "unmatched"
}
