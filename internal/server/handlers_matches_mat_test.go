package server

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/kevung/gnubgparser"
)

// TestExportMatchMATRoute imports a .mat, exports it back through
// /v1/matches.exportMat, and checks the response is a text/plain .mat that
// re-parses to the same number of games.
func TestExportMatchMATRoute(t *testing.T) {
	ts := newTestServer(t)

	fixture, err := os.ReadFile("../../testdata/test.mat")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	events := uploadImportNamed(t, ts, "/v1/imports.gnubg", "test.mat", fixture)
	done := events[len(events)-1]
	if done["event"] != "done" {
		t.Fatalf("last event = %v, want done", done["event"])
	}
	matchID := int64(done["match_id"].(float64))
	if matchID == 0 {
		t.Fatal("match_id = 0")
	}

	resp := post(t, ts, "/v1/matches.exportMat", map[string]any{"matchId": matchID})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("export status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("content-type = %q, want text/plain", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "point match") {
		t.Fatalf("body is not a .mat:\n%s", body)
	}

	parsed, err := gnubgparser.ParseMAT(strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("exported .mat does not re-parse: %v", err)
	}
	orig, _ := gnubgparser.ParseMATFile("../../testdata/test.mat")
	if len(parsed.Games) != len(orig.Games) {
		t.Errorf("exported games = %d, want %d", len(parsed.Games), len(orig.Games))
	}
}

// TestExportMatchMATUnknown: exporting a missing match id is an error, not a panic.
func TestExportMatchMATUnknown(t *testing.T) {
	ts := newTestServer(t)
	resp := post(t, ts, "/v1/matches.exportMat", map[string]any{"matchId": 999999})
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Errorf("unknown match exported 200, want an error status")
	}
}
