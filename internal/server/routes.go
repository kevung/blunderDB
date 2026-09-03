package server

import (
	"net/http"
	"sort"
	"strings"
)

// route is one entry in the server's routing table: an HTTP method, a
// net/http pattern (Go 1.22+ method-aware patterns), and its handler.
type route struct {
	method  string
	pattern string
	handler http.HandlerFunc
}

// routes returns the full routing table. Ops endpoints (health, readiness,
// metrics) are always present; /metrics is gated on EnableMetrics. The domain
// surface (POST /v1/<family>.<method>) is contributed by domainRoutes.
func (s *Server) routes() []route {
	rs := []route{
		{http.MethodGet, "/healthz", s.health.Live},
		{http.MethodGet, "/readyz", s.health.Ready},
	}
	if s.opts.EnableMetrics {
		rs = append(rs, route{http.MethodGet, "/metrics", s.health.Expose})
	}
	rs = append(rs, s.domainRoutes()...)
	return rs
}

// domainRoutes returns the /v1 domain handlers, one group per storage family.
func (s *Server) domainRoutes() []route {
	var rs []route
	rs = append(rs, s.positionRoutes()...)
	rs = append(rs, s.analysisRoutes()...)
	rs = append(rs, s.matchRoutes()...)
	rs = append(rs, s.commentRoutes()...)
	rs = append(rs, s.collectionRoutes()...)
	rs = append(rs, s.tournamentRoutes()...)
	rs = append(rs, s.ankiRoutes()...)
	rs = append(rs, s.filterRoutes()...)
	rs = append(rs, s.sessionRoutes()...)
	rs = append(rs, s.searchRoutes()...)
	rs = append(rs, s.metadataRoutes()...)
	rs = append(rs, s.statsRoutes()...)
	rs = append(rs, s.ingestRoutes()...)
	rs = append(rs, s.tenantRoutes()...)
	rs = append(rs, s.maintenanceRoutes()...)
	rs = append(rs, s.gammonnetRoutes()...)
	return rs
}

// Paths returns the sorted list of registered /v1 domain route patterns
// (e.g. "/v1/positions.save"). It backs the `call --list` dispatcher and lets
// callers enumerate the full Storage surface exposed over HTTP.
func (s *Server) Paths() []string {
	var out []string
	for _, rt := range s.domainRoutes() {
		if strings.HasPrefix(rt.pattern, "/v1/") {
			out = append(out, rt.pattern)
		}
	}
	sort.Strings(out)
	return out
}

// notFound writes the API error envelope for an unmatched route. It is the
// catch-all so clients always receive the documented {"error":{...}} shape
// rather than net/http's plain-text 404.
func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	writeErrorCode(w, CodeNotFound, "unknown route: "+r.Method+" "+r.URL.Path)
}

// methodNotAllowed wraps mux so a request to a KNOWN path called with the
// wrong method answers the API's own JSON error envelope — 405, with an
// Allow header naming the one method the path accepts — instead of
// net/http's built-in text/plain "405 method not allowed" (Go's 1.22+
// method-aware ServeMux already produces that automatically; this only
// changes its shape to match every other error response on this daemon)
// (#232).
func (s *Server) methodNotAllowed(mux http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want, ok := s.allowedMethod[r.URL.Path]; ok && r.Method != want {
			writeMethodNotAllowed(w, want)
			return
		}
		mux.ServeHTTP(w, r)
	})
}
