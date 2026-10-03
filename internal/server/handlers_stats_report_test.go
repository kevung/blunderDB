package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestStatsReportRoute: the daemon answers the report the CLI writes, in the
// requested language, from the same generator.
func TestStatsReportRoute(t *testing.T) {
	ts := newTestServer(t)
	resp := post(t, ts, "/v1/stats.report", statsReportReq{Filter: storage.StatsFilter{DecisionType: -1}, Language: "fr"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out statsReportResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<!doctype html>", `lang="fr"`, "Rapport blunderDB"} {
		if !strings.Contains(out.HTML, want) {
			t.Errorf("report lacks %q", want)
		}
	}
}
