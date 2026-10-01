package server

import (
	"strings"
	"testing"
)

// TestRoutes_EveryServedRouteIsAccountedFor: every route mounted under /v1/ or /ops/ is either
// one Paths() lists — and so one `call` serves and TestDatabaseParity checks — or one
// serverOnlyPaths names with its reason. A route in neither would escape the parity rule.
func TestRoutes_EveryServedRouteIsAccountedFor(t *testing.T) {
	_, srv := eventServerOn(t, memStorage(t), nil)
	listed := map[string]bool{}
	for _, p := range srv.Paths() {
		listed[p] = true
	}
	mounted := map[string]bool{}
	for _, rt := range srv.routes() {
		if !strings.HasPrefix(rt.pattern, "/v1/") && !strings.HasPrefix(rt.pattern, "/ops/") {
			continue
		}
		mounted[rt.pattern] = true
		if !listed[rt.pattern] && serverOnlyPaths[rt.pattern] == "" {
			t.Errorf("%s is served but neither in Paths() nor in serverOnlyPaths", rt.pattern)
		}
	}
	for p, why := range serverOnlyPaths {
		switch {
		case strings.TrimSpace(why) == "":
			t.Errorf("serverOnlyPaths gives no reason for %s", p)
		case listed[p]:
			t.Errorf("%s is in Paths() and in serverOnlyPaths", p)
		case !mounted[p]:
			t.Errorf("serverOnlyPaths names %s, which is not served", p)
		}
	}
}
