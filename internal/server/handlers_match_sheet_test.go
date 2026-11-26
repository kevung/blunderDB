package server

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/printsheet"
)

// sheetHTML posts body to matches.sheet and returns the page.
func sheetHTML(t *testing.T, body any) string {
	t.Helper()
	ts := newTestServer(t)
	fixture, err := os.ReadFile("../../testdata/test.mat")
	if err != nil {
		t.Fatal(err)
	}
	events := uploadImportNamed(t, ts, "/v1/imports.gnubg", "test.mat", fixture)
	id := int64(events[len(events)-1]["match_id"].(float64))
	b, _ := json.Marshal(body)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	m["matchIds"] = []int64{id}
	resp := post(t, ts, "/v1/matches.sheet", m)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out matchSheetResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out.HTML
}

const archiveMarker = `<script type="application/vnd.blunderdb.archive">`

// TestMatchSheetRouteDefaults: without options the daemon prints what the
// dialog and the CLI print by default, the reimport code among them.
func TestMatchSheetRouteDefaults(t *testing.T) {
	if !strings.Contains(sheetHTML(t, map[string]any{}), archiveMarker) {
		t.Error("no options: the reimport code is missing, defaults not applied")
	}
}

// TestMatchSheetRoutePartialOptions: an options object naming one choice
// keeps the defaults of the others.
func TestMatchSheetRoutePartialOptions(t *testing.T) {
	if !strings.Contains(sheetHTML(t, map[string]any{"options": map[string]any{"expanded": true}}), archiveMarker) {
		t.Error("partial options: the reimport code default was lost")
	}
	if strings.Contains(sheetHTML(t, map[string]any{"options": map[string]any{"reimportCode": false}}), archiveMarker) {
		t.Error("reimportCode false: the code is still printed")
	}
}

// TestMatchSheetRouteCap: more than MaxMatches is refused, not rendered.
func TestMatchSheetRouteCap(t *testing.T) {
	ts := newTestServer(t)
	ids := make([]int64, printsheet.MaxMatches+1)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	resp := post(t, ts, "/v1/matches.sheet", map[string]any{"matchIds": ids})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}
