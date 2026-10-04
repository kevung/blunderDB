package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestTrainingJournalRoundTrip: a session saved over HTTP is read back by
// training.sessions and folded into training.numberStats, as the desktop tab
// reads it from the same store.
func TestTrainingJournalRoundTrip(t *testing.T) {
	ts := newTestServer(t)
	session := storage.TrainingSession{
		Exercise: "pips", SeedSource: "pool", NumbersAsked: 2, Faults: 1,
		Items: []storage.TrainingItem{{NumberType: "pips.bottom", Wrong: true}, {NumberType: "pips.top"}},
	}
	var saved trainingSaveResp
	postDecode(t, ts, "/v1/training.save", session, &saved)
	if saved.ID == 0 {
		t.Fatal("training.save returns the new session's id")
	}

	var sessions []storage.TrainingSession
	postDecode(t, ts, "/v1/training.sessions", trainingSessionsReq{Exercise: "pips"}, &sessions)
	if len(sessions) != 1 || sessions[0].ID != saved.ID || sessions[0].Faults != 1 {
		t.Fatalf("training.sessions = %+v; want the saved session", sessions)
	}
	var stats []storage.TrainingNumberStat
	postDecode(t, ts, "/v1/training.numberStats", trainingStatsReq{Exercise: "pips"}, &stats)
	if len(stats) != 2 {
		t.Fatalf("training.numberStats = %+v; want one row per number type", stats)
	}

	var none []storage.TrainingSession
	postDecode(t, ts, "/v1/training.sessions", trainingSessionsReq{Exercise: "other"}, &none)
	if none == nil || len(none) != 0 {
		t.Errorf("an exercise without sessions is an empty list, not null: %+v", none)
	}

	for _, bad := range []struct {
		path string
		body any
	}{
		{"/v1/training.save", storage.TrainingSession{}},
		{"/v1/training.sessions", trainingSessionsReq{Limit: -1}},
	} {
		r := post(t, ts, bad.path, bad.body)
		r.Body.Close()
		if r.StatusCode != http.StatusBadRequest {
			t.Errorf("%s %+v: status %d; want 400", bad.path, bad.body, r.StatusCode)
		}
	}
}

func postDecode(t *testing.T, ts *httptest.Server, path string, body, v any) {
	t.Helper()
	resp := post(t, ts, path, body)
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %.300s", resp.StatusCode, b)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("decode %.300s: %v", b, err)
	}
}

// TestGammonNetEvaluate: a bare XGID is evaluated without touching the
// tenant, and an out-of-range depth is refused before any search.
func TestGammonNetEvaluate(t *testing.T) {
	ts := newTestServer(t)
	zero := 0
	var out gammonnetEvaluateResp
	postDecode(t, ts, "/v1/gammonnet.evaluate", gammonnetEvaluateReq{
		XGID: "XGID=-b----E-C---eE---c-e----B-:0:0:1:52:0:0:0:0:10", Ply: &zero, Candidates: 3,
	}, &out)
	if out.Decision != "checker" || len(out.Moves) == 0 || len(out.Moves) > 3 || out.Depth == "" {
		t.Fatalf("gammonnet.evaluate = %+v", out)
	}
	deep := 5
	r := post(t, ts, "/v1/gammonnet.evaluate", gammonnetEvaluateReq{XGID: "XGID=-b----E-C---eE---c-e----B-:0:0:1:52:0:0:0:0:10", Ply: &deep})
	r.Body.Close()
	if r.StatusCode != http.StatusBadRequest {
		t.Errorf("ply 5: status %d; want 400", r.StatusCode)
	}
}

// TestGammonNetEvaluateUsesTheTenantsMET: a bare evaluation at a match score
// is valued with the tenant's current table, as its stored analyses are
// (ADR-0068).
func TestGammonNetEvaluateUsesTheTenantsMET(t *testing.T) {
	ts := newTestServer(t)
	zero := 0
	req := gammonnetEvaluateReq{XGID: "XGID=-b----E-C---eE---c-e----B-:0:0:1:00:2:4:0:7:10", Ply: &zero}
	var builtIn, club gammonnetEvaluateResp
	postDecode(t, ts, "/v1/gammonnet.evaluate", req, &builtIn)
	if builtIn.Cube == nil {
		t.Fatalf("gammonnet.evaluate = %+v; want a cube decision", builtIn)
	}

	source, err := os.ReadFile("../../pkg/blunderdb/engine/testdata/met/Rockwell-Kazaross.xml")
	if err != nil {
		t.Fatal(err)
	}
	var table domain.MatchEquityTable
	postDecode(t, ts, "/v1/met.import", map[string]any{"source": string(source)}, &table)
	var ok struct{}
	postDecode(t, ts, "/v1/met.setCurrent", map[string]any{"id": table.ID}, &ok)

	postDecode(t, ts, "/v1/gammonnet.evaluate", req, &club)
	if club.Cube == nil || club.Cube.CubefulNoDoubleEquity == builtIn.Cube.CubefulNoDoubleEquity {
		t.Errorf("no-double equity %v under the club table, %v under the built-in one; want them to differ",
			club.Cube, builtIn.Cube.CubefulNoDoubleEquity)
	}
}
