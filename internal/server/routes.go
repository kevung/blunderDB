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
	return rs
}

// notFound writes the API error envelope for an unmatched route. It is the
// catch-all so clients always receive the documented {"error":{...}} shape
// rather than net/http's plain-text 404.
func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	writeErrorCode(w, CodeNotFound, "unknown route: "+r.Method+" "+r.URL.Path)
}
