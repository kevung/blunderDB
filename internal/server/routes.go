package server

import "net/http"

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

// domainRoutes returns the /v1 domain handlers. PR2 fills this in with the
// per-family handlers (positions.*, analyses.*, matches.*, …). PR1 ships the
// skeleton, so the slice is empty and unmatched /v1 paths fall through to the
// 404 envelope.
func (s *Server) domainRoutes() []route {
	return nil
}

// notFound writes the API error envelope for an unmatched route. It is the
// catch-all so clients always receive the documented {"error":{...}} shape
// rather than net/http's plain-text 404.
func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	writeErrorCode(w, CodeNotFound, "unknown route: "+r.Method+" "+r.URL.Path)
}
