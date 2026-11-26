package server

import (
	"net/http"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

func TestGammonNetScoreMap(t *testing.T) {
	ts := newTestServer(t)
	zero := 0
	var m gammonnet.ScoreMap
	postDecode(t, ts, "/v1/gammonnet.scoreMap", scoreMapReq{
		XGID: "XGID=-b----E-C---eE---c-e----B-:0:0:1:31:6:4:0:9:10", MatchLength: 3, Ply: &zero,
	}, &m)
	if m.Decision != "checker" || len(m.Cells) != 9 || m.Cells[0].Move == "" {
		t.Fatalf("scoreMap = %+v", m)
	}

	deep := 3
	for name, req := range map[string]scoreMapReq{
		"even length": {XGID: rollsXGID, MatchLength: 8},
		"too long":    {XGID: rollsXGID, MatchLength: 17},
		"too deep":    {XGID: rollsXGID, Ply: &deep},
		"no position": {},
	} {
		r := post(t, ts, "/v1/gammonnet.scoreMap", req)
		r.Body.Close()
		if r.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", name, r.StatusCode)
		}
	}
}
