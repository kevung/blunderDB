package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestStatsSummaryRoute: the daemon answers the public summary, whose JSON
// carries neither TopBlunders nor PerMatch.
func TestStatsSummaryRoute(t *testing.T) {
	ts := newTestServer(t)
	resp := post(t, ts, "/v1/stats.summary", statsComputeReq{Filter: storage.StatsFilter{DecisionType: -1}})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"TopBlunders", "PerMatch", "PerTournament", "PerTag"} {
		if _, ok := out[leak]; ok {
			t.Errorf("summary carries %s", leak)
		}
	}
	if _, ok := out["PerPhase"]; !ok {
		t.Errorf("summary lacks PerPhase: %v", out)
	}
}
