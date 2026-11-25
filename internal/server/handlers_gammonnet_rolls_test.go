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
