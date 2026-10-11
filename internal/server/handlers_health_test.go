package server

import (
	"bufio"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/maintenance"
)

// The state of health over HTTP (ADR-0088): status lists every line, run
// streams started then done and records the run, which the next status
// reads back.
func TestMaintenanceRun_StreamsAndRecords(t *testing.T) {
	ts, _ := newTestServerAndHandler(t)

	resp := post(t, ts, "/v1/maintenance.run", map[string]any{"kind": "stats.matches"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var events []string
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		var ev struct {
			Event string `json:"event"`
		}
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			t.Fatalf("not NDJSON: %s", sc.Bytes())
		}
		events = append(events, ev.Event)
	}
	if len(events) < 2 || events[0] != "started" || events[len(events)-1] != "done" {
		t.Fatalf("events %v, want started … done", events)
	}

	st := post(t, ts, "/v1/maintenance.status", map[string]any{})
	defer st.Body.Close()
	var lines []maintenance.Line
	if err := json.NewDecoder(st.Body).Decode(&lines); err != nil {
		t.Fatal(err)
	}
	if len(lines) != len(maintenance.Kinds()) {
		t.Fatalf("%d lines, want %d", len(lines), len(maintenance.Kinds()))
	}
	for _, l := range lines {
		if l.Kind == maintenance.StatsMatches && (l.Last == nil || l.Last.Outcome != maintenance.OutcomeDone) {
			t.Errorf("stats.matches last run %+v, want done", l.Last)
		}
	}
}

// A maintenance run takes the tenant's one slot, shared with the gammonNet
// sweeps: refused with a plain 409 while it is held, before any stream.
func TestMaintenanceRun_SharesTheTenantSlot(t *testing.T) {
	ts, srv := newTestServerAndHandler(t)
	held, err := srv.gammonnetJobs.startExclusive(testTenant, func() {})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.gammonnetJobs.finish(held)
	resp := post(t, ts, "/v1/maintenance.run", map[string]any{"kind": "stats.matches"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status %d, want 409", resp.StatusCode)
	}
}

func TestMaintenanceRefusesUnknownAndCountless(t *testing.T) {
	ts, _ := newTestServerAndHandler(t)
	for path, kind := range map[string]string{
		"/v1/maintenance.run":   "nope",
		"/v1/maintenance.count": "analysis.columns",
		"/v1/maintenance.ids":   "trash",
	} {
		resp := post(t, ts, path, map[string]any{"kind": kind})
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s %s: status %d, want 400", path, kind, resp.StatusCode)
		}
	}
}
