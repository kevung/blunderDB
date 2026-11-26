package server

import (
	"net/http"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

const rollsXGID = "XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:0:10"

func TestGammonNetRolls(t *testing.T) {
	ts := newTestServer(t)
	zero := 0
	var m gammonnet.RollMap
	postDecode(t, ts, "/v1/gammonnet.rolls", rollsReq{XGID: rollsXGID, Ply: &zero}, &m)
	if len(m.Rows) != gammonnet.NumRolls || m.Ply != 0 || !m.CubeAccess {
		t.Fatalf("rolls = %+v", m)
	}

	// The replies to a play: the opponent rolls, with the cube still centred.
	var replies gammonnet.RollMap
	postDecode(t, ts, "/v1/gammonnet.rolls", rollsReq{XGID: rollsXGID, Ply: &zero, Dice: &[2]int{3, 1}, Move: "8/5 6/5"}, &replies)
	if len(replies.Rows) != gammonnet.NumRolls {
		t.Fatalf("replies = %+v", replies)
	}

	deep := 3
	for name, req := range map[string]rollsReq{
		"too deep":     {XGID: rollsXGID, Ply: &deep},
		"illegal play": {XGID: rollsXGID, Ply: &zero, Dice: &[2]int{3, 1}, Move: "24/18"},
		"move no dice": {XGID: rollsXGID, Ply: &zero, Move: "8/5 6/5"},
		"no position":  {},
	} {
		r := post(t, ts, "/v1/gammonnet.rolls", req)
		r.Body.Close()
		if r.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", name, r.StatusCode)
		}
	}
}

// The daemon's maps are cached per tenant, rolls and scores alike: a tenant
// asking again is served, another tenant asking the same is searched, and
// learns nothing from it.
func TestGammonNetMaps_CachePerTenant(t *testing.T) {
	zero := 0
	for path, req := range map[string]any{
		"/v1/gammonnet.rolls":    rollsReq{XGID: rollsXGID, Ply: &zero, PruneK: 5},
		"/v1/gammonnet.scoreMap": scoreMapReq{XGID: "XGID=-b----E-C---eE---c-e----B-:0:0:1:31:6:4:0:9:10", MatchLength: 3, Ply: &zero},
	} {
		ts, srv := newTestServerAndHandler(t)
		srv.maps = gammonnet.NewMapCache(gammonnet.DefaultMapCacheBytes)
		ask := func(tenant string) int {
			t.Helper()
			r := postAsTenant(t, ts, tenant, path, req)
			r.Body.Close()
			if r.StatusCode != http.StatusOK {
				t.Fatalf("%s tenant %s: status %d", path, tenant, r.StatusCode)
			}
			return srv.maps.Len()
		}
		if first := ask("1"); first != 1 {
			t.Fatalf("%s: %d entries after the first map, want 1", path, first)
		}
		if again := ask("1"); again != 1 {
			t.Fatalf("%s: the same tenant asking again left %d entries, want 1", path, again)
		}
		if other := ask("2"); other != 2 {
			t.Fatalf("%s: %d entries after another tenant, want its own: 2", path, other)
		}
	}
}
