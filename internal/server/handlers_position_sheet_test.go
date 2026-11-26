package server

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/printsheet"
)

// positionSheet imports a match and posts body to positions.sheet.
func positionSheet(t *testing.T, body map[string]any) (int, string) {
	t.Helper()
	ts := newTestServer(t)
	fixture, err := os.ReadFile("../../testdata/test.mat")
	if err != nil {
		t.Fatal(err)
	}
	uploadImportNamed(t, ts, "/v1/imports.gnubg", "test.mat", fixture)
	resp := post(t, ts, "/v1/positions.sheet", body)
	defer resp.Body.Close()
	var out matchSheetResp
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
	}
	return resp.StatusCode, out.HTML
}

// TestPositionSheetRouteDefaults: a query without options prints the whole
// result with the defaults of the dialog and the CLI — the analysis on the
// line — and the options object keeps the defaults it does not name.
func TestPositionSheetRouteDefaults(t *testing.T) {
	code, page := positionSheet(t, map[string]any{"query": "s"})
	if code != http.StatusOK || !strings.Contains(page, `class="dl two"`) {
		t.Fatalf("status %d, analysis on the line %v", code, strings.Contains(page, `class="dl two"`))
	}
	code, page = positionSheet(t, map[string]any{"query": "s", "options": map[string]any{"verso": true}})
	if code != http.StatusOK || !strings.Contains(page, "<div data-verso>") {
		t.Errorf("verso: status %d, back present %v", code, strings.Contains(page, "<div data-verso>"))
	}
}

// TestPositionSheetRouteRefusals: too long a list, or none, is a 400.
func TestPositionSheetRouteRefusals(t *testing.T) {
	ids := make([]int64, printsheet.MaxPositions+1)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	for _, body := range []map[string]any{{"ids": ids}, {}} {
		if code, _ := positionSheet(t, body); code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", code)
		}
	}
}

// TestPositionSheetReimportRoute: a list printed with its reimport codes
// imports into another library through imports.archive, its positions saved
// on their own, and a second pass finds them all held.
func TestPositionSheetReimportRoute(t *testing.T) {
	code, page := positionSheet(t, map[string]any{"query": "s", "options": map[string]any{"reimportCode": true}})
	if code != http.StatusOK || !strings.Contains(page, `class="qrb" data-from="1"`) {
		t.Fatalf("status %d, code drawn %v", code, strings.Contains(page, `data-from="1"`))
	}
	dst := newTestServer(t)
	done := uploadImportNamed(t, dst, "/v1/imports.archive", "liste.html", []byte(page))
	last := done[len(done)-1]
	n, _ := last["list_positions"].(float64)
	if last["event"] != "done" || n == 0 {
		t.Fatalf("list import: %v", last)
	}
	again := uploadImportNamed(t, dst, "/v1/imports.archive", "liste.html", []byte(page))
	if last := again[len(again)-1]; last["event"] != "done" || last["list_positions"].(float64) != 0 || last["list_positions_known"].(float64) != n {
		t.Fatalf("second list import: %v", last)
	}
}
