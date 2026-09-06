package server

import (
	"net/http"
	"reflect"
	"runtime"
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

// handlerKind is inferred from the name of the function that produced the
// http.HandlerFunc: rpc/rpcVoid/rpcStream are generic builders whose closures
// carry the builder's name (server.rpcStream[...].func1); everything else is
// a hand-written handler. Two things read this classification: the routing
// smoke test's Content-Type check (routes_smoke_test.go), and
// streamingPaths (server.go), which gives a streaming-shaped route a longer
// read/write deadline than an ordinary one (#234) — one classification, so
// the two can never quietly drift apart.
type handlerKind int

const (
	kindJSON handlerKind = iota
	kindStream
	kindCustom
)

func kindOf(h http.HandlerFunc) handlerKind {
	name := runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()
	switch {
	case strings.Contains(name, ".rpcStream["):
		return kindStream
	case strings.Contains(name, ".rpc["), strings.Contains(name, ".rpcVoid["):
		return kindJSON
	default:
		return kindCustom
	}
}

// streamingCustomPaths names the hand-written (kindCustom) handlers that
// stream a response or can otherwise run long — file uploads/downloads and
// the gammonNet sweeps — as opposed to the ones that are a small, quick JSON
// call in and out (imports.cancel, gammonnet.*.cancel, ops/tenant.purge,
// ops/maintenance.vacuum, matches.exportMat) despite not being built by
// rpc/rpcVoid. Kept exhaustive on purpose, like routes_smoke_test.go's
// customContentTypes: a new streaming custom handler is invisible to
// kindOf's reflection trick and must be added here explicitly.
var streamingCustomPaths = map[string]bool{
	"/v1/imports.json":             true,
	"/v1/imports.xg":               true,
	"/v1/imports.gnubg":            true,
	"/v1/imports.bgf":              true,
	"/v1/imports.db":               true,
	"/v1/imports.position":         true,
	"/v1/exports.json":             true,
	"/v1/exports.sqlite":           true,
	"/v1/gammonnet.analyzeMissing": true,
	"/v1/gammonnet.sweepStale":     true,
	// search.query streams a whole result set, exactly like the rpcStream
	// search.find it delegates to; it is hand-written only to refuse an
	// unreadable query before the 200 is committed (B.18, #186).
	"/v1/search.query": true,
}

// routes returns the full routing table served on the main listener. Health,
// readiness and metrics are always present (/metrics gated on EnableMetrics);
// the domain surface (POST /v1/<family>.<method>) is contributed by
// domainRoutes. The /ops/ family joins it only when no separate ops listener
// is configured — see opsRoutes and OpsAddr.
func (s *Server) routes() []route {
	rs := []route{
		{http.MethodGet, "/healthz", s.health.Live},
		{http.MethodGet, "/readyz", s.health.Ready},
	}
	if s.opts.EnableMetrics {
		rs = append(rs, route{http.MethodGet, "/metrics", s.health.Expose})
	}
	rs = append(rs, s.domainRoutes()...)
	if s.opts.OpsAddr == "" {
		rs = append(rs, s.opsRoutes()...)
	}
	return rs
}

// opsRoutes returns the operator surface: the calls whose effect does not stop
// at the caller's own tenant.
//
// The daemon authenticates nobody (ADR-0005) — it trusts X-Tenant-ID and runs
// behind a proxy. Under that rule, a route reachable by any tenant is a route
// every tenant may call, and two of them were never tenant-shaped:
//
//   - maintenance.vacuum rewrites the whole SQLite file, all tenants' data
//     included, and holds a write lock for the duration;
//   - tenant.purge destroys a tenant's data, and the tenant it destroys is the
//     one named in the header the caller controls.
//
// Under /ops/ they are trivially excluded by a proxy (one prefix), and with
// --ops-addr they are not even served on the public listener. What did NOT
// move: gammonnet.sweepStale, which the fiche listed as "the whole file" but
// which is scoped — gammonnetPositionsWithStaleAnalysis drains the caller's
// scope, nothing else. It is expensive, not cross-tenant; the rate limiter and
// the in-flight gauges are what bound it.
func (s *Server) opsRoutes() []route {
	var rs []route
	rs = append(rs, s.maintenanceRoutes()...)
	rs = append(rs, s.tenantRoutes()...)
	return rs
}

// opsPaths is the set of patterns opsRoutes serves, for the tests and for
// callers that need to tell the two surfaces apart.
func (s *Server) opsPaths() map[string]bool {
	out := make(map[string]bool)
	for _, rt := range s.opsRoutes() {
		out[rt.pattern] = true
	}
	return out
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
	rs = append(rs, s.gammonnetRoutes()...)
	rs = append(rs, s.trashRoutes()...)
	return rs
}

// streamingPaths is the set of registered patterns whose handler streams a
// response or can otherwise run long: every rpcStream route, plus the
// hand-written ones streamingCustomPaths names. server.go's methodDeadline
// gives these a longer read/write deadline than an ordinary request/response
// round-trip gets (#234).
func (s *Server) streamingPaths() map[string]bool {
	out := make(map[string]bool)
	for _, rt := range s.routes() {
		switch kindOf(rt.handler) {
		case kindStream:
			out[rt.pattern] = true
		case kindCustom:
			if streamingCustomPaths[rt.pattern] {
				out[rt.pattern] = true
			}
		}
	}
	return out
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
	// The ops family is listed too. `call` is an in-process dispatcher, not a
	// network surface: separating vacuum and purge from the tenant routes is
	// about what a proxy exposes, and dropping them from `call --list` would
	// take two capabilities away from the CLI for a reason that has nothing to
	// do with it (CLAUDE.md's CLI/GUI/server parity).
	for _, rt := range s.opsRoutes() {
		out = append(out, rt.pattern)
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
