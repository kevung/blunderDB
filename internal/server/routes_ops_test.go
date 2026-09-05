package server

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// The operator surface (G.5, #233). What is pinned here is the frontier: which
// calls sit behind /ops/, that they still demand a tenant, and that asking for
// a separate listener takes them off the public one entirely.

func opsServer(t *testing.T, opsAddr string) *Server {
	t.Helper()
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	srv, err := New(Options{Storage: st, OpsAddr: opsAddr, Logger: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatal(err)
	}
	return srv
}

func TestOps_HoldsTheCallsThatLeaveTheTenant(t *testing.T) {
	srv := opsServer(t, "")
	got := srv.opsPaths()
	want := map[string]bool{
		"/ops/maintenance.vacuum": true, // rewrites the whole file
		"/ops/tenant.purge":       true, // destroys the tenant it is handed
	}
	if len(got) != len(want) {
		t.Errorf("ops routes = %v, want %v", got, want)
	}
	for p := range want {
		if !got[p] {
			t.Errorf("%s is not in the ops family", p)
		}
	}
	// The sweep is expensive but scoped — gammonnetPositionsWithStaleAnalysis
	// drains the caller's scope and nothing else — so it stays a tenant route.
	for _, rt := range srv.domainRoutes() {
		if rt.pattern == "/v1/gammonnet.sweepStale" {
			return
		}
	}
	t.Error("gammonnet.sweepStale left the tenant surface; it is scoped, and belongs there")
}

func TestOps_StillRequiresATenant(t *testing.T) {
	srv := opsServer(t, "")
	for path := range srv.opsPaths() {
		if srv.publicPaths()[path] {
			t.Errorf("%s is reachable with no tenant; a purge names the tenant it destroys", path)
		}
	}
	// And the probes, which have no tenant to speak of, still are public.
	for _, p := range []string{"/healthz", "/readyz"} {
		if !srv.publicPaths()[p] {
			t.Errorf("%s must stay reachable without a tenant", p)
		}
	}
}

func TestOps_SeparateListenerTakesThemOffThePublicOne(t *testing.T) {
	shared := opsServer(t, "")
	for path := range shared.opsPaths() {
		if !shared.knownPaths[path] {
			t.Errorf("with no --ops-addr, %s must still be served on the main listener", path)
		}
	}

	split := opsServer(t, "127.0.0.1:0")
	main := map[string]bool{}
	for _, rt := range split.routes() {
		main[rt.pattern] = true
	}
	for path := range split.opsPaths() {
		if main[path] {
			t.Errorf("with --ops-addr set, %s must NOT be on the main listener", path)
		}
	}
	if split.opsHTTP == nil {
		t.Fatal("no ops listener was built")
	}
	// They remain callable — through the ops server, and through `call`.
	for path := range split.opsPaths() {
		if !split.knownPaths[path] {
			t.Errorf("%s is unknown to the server; its 404 would be the catch-all's", path)
		}
	}
	var listed []string
	for _, p := range split.Paths() {
		if strings.HasPrefix(p, "/ops/") {
			listed = append(listed, p)
		}
	}
	if len(listed) != len(split.opsPaths()) {
		t.Errorf("Paths() lists %v of the ops family; `call` needs all of it", listed)
	}
}

func TestOps_AreStillPOSTOnly(t *testing.T) {
	srv := opsServer(t, "")
	for _, rt := range srv.opsRoutes() {
		if rt.method != http.MethodPost {
			t.Errorf("%s uses %s; the surface is POST-only", rt.pattern, rt.method)
		}
	}
}
