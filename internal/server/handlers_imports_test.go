package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// exportJSON returns the NDJSON export of the server's current state.
func exportJSON(t *testing.T, ts *httptest.Server) []byte {
	t.Helper()
	resp := post(t, ts, "/v1/exports.json", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("export status = %d, want 200", resp.StatusCode)
	}
	b, _ := io.ReadAll(resp.Body)
	return b
}

// uploadImport posts a multipart file (named data.ndjson) to an import endpoint
// and returns the NDJSON event lines.
func uploadImport(t *testing.T, ts *httptest.Server, path string, file []byte) []map[string]any {
	return uploadImportNamed(t, ts, path, "data.ndjson", file)
}

// uploadImportNamed is uploadImport with an explicit upload filename, so
// extension-dispatched formats (GnuBG .sgf vs .mat) reach the right parser.
func uploadImportNamed(t *testing.T, ts *httptest.Server, path, filename string, file []byte) []map[string]any {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(file); err != nil {
		t.Fatal(err)
	}
	mw.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+path, &body)
	req.Header.Set(middleware.TenantHeader, testTenant)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("import status = %d, want 200", resp.StatusCode)
	}

	var events []map[string]any
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		line := sc.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var ev map[string]any
		if err := json.Unmarshal(line, &ev); err != nil {
			t.Fatalf("event line not JSON: %q", line)
		}
		events = append(events, ev)
	}
	return events
}

func TestImportExportJSONRoundtrip(t *testing.T) {
	ts := newTestServer(t)

	// Seed one position with a comment.
	p := domain.InitializePosition()
	saveResp := post(t, ts, "/v1/positions.save", positionReq{Position: &p})
	var saved idResp
	json.NewDecoder(saveResp.Body).Decode(&saved)
	saveResp.Body.Close()
	post(t, ts, "/v1/comments.add", commentAddReq{PositionID: saved.ID, Text: "hello"}).Body.Close()

	// Export, then re-import into the same DB (dedup keeps positions at 1).
	dump := exportJSON(t, ts)
	if !bytes.Contains(dump, []byte("hello")) {
		t.Fatalf("export missing comment:\n%s", dump)
	}

	events := uploadImport(t, ts, "/v1/imports.json", dump)
	if len(events) < 2 {
		t.Fatalf("expected >=2 events, got %d: %v", len(events), events)
	}
	if events[0]["event"] != "started" {
		t.Fatalf("first event = %v, want started", events[0]["event"])
	}
	last := events[len(events)-1]
	if last["event"] != "done" {
		t.Fatalf("last event = %v, want done", last["event"])
	}
	if last["saved_positions"].(float64) != 1 {
		t.Fatalf("saved_positions = %v, want 1", last["saved_positions"])
	}
	if _, ok := events[0]["import_id"].(string); !ok {
		t.Fatal("started event missing import_id")
	}
}

func TestImportXGEndToEnd(t *testing.T) {
	ts := newTestServer(t)

	fixture, err := os.ReadFile(filepath.Join("..", "..", "testdata", "match_with_comment.xg"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	events := uploadImport(t, ts, "/v1/imports.xg", fixture)
	if len(events) < 2 {
		t.Fatalf("expected >=2 events, got %d: %v", len(events), events)
	}
	if events[0]["event"] != "started" {
		t.Fatalf("first event = %v, want started", events[0]["event"])
	}
	done := events[len(events)-1]
	if done["event"] != "done" {
		t.Fatalf("last event = %v, want done", done["event"])
	}
	if done["matches"].(float64) != 1 {
		t.Fatalf("matches = %v, want 1", done["matches"])
	}
	savedFirst := done["saved_positions"].(float64)
	if savedFirst == 0 {
		t.Fatalf("saved_positions = 0, want > 0")
	}
	matchID := done["match_id"].(float64)
	if matchID == 0 {
		t.Fatal("match_id = 0, want > 0")
	}

	// A second identical upload must dedup: skipped, same match id, no new positions.
	events2 := uploadImport(t, ts, "/v1/imports.xg", fixture)
	done2 := events2[len(events2)-1]
	if done2["skipped_duplicates"].(float64) != 1 {
		t.Fatalf("skipped_duplicates = %v, want 1", done2["skipped_duplicates"])
	}
	if done2["saved_positions"].(float64) != 0 {
		t.Fatalf("saved_positions on dup = %v, want 0", done2["saved_positions"])
	}
	if done2["match_id"].(float64) != matchID {
		t.Fatalf("dup match_id = %v, want %v", done2["match_id"], matchID)
	}
}

func TestImportGnuBGEndToEnd(t *testing.T) {
	ts := newTestServer(t)

	fixture, err := os.ReadFile(filepath.Join("..", "..", "testdata", "test.sgf"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	events := uploadImportNamed(t, ts, "/v1/imports.gnubg", "test.sgf", fixture)
	done := events[len(events)-1]
	if done["event"] != "done" {
		t.Fatalf("last event = %v, want done", done["event"])
	}
	if done["matches"].(float64) != 1 {
		t.Fatalf("matches = %v, want 1", done["matches"])
	}
	if done["saved_positions"].(float64) == 0 {
		t.Fatal("saved_positions = 0, want > 0")
	}

	// Second identical upload dedups.
	done2 := func() map[string]any {
		evs := uploadImportNamed(t, ts, "/v1/imports.gnubg", "test.sgf", fixture)
		return evs[len(evs)-1]
	}()
	if done2["skipped_duplicates"].(float64) != 1 {
		t.Fatalf("skipped_duplicates = %v, want 1", done2["skipped_duplicates"])
	}
}

func TestImportBGFEndToEnd(t *testing.T) {
	ts := newTestServer(t)

	fixture, err := os.ReadFile(filepath.Join("..", "..", "testdata", "TachiAI_V_player_Nov_2__2025__16_55.bgf"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	events := uploadImportNamed(t, ts, "/v1/imports.bgf", "match.bgf", fixture)
	done := events[len(events)-1]
	if done["event"] != "done" {
		t.Fatalf("last event = %v, want done", done["event"])
	}
	if done["matches"].(float64) != 1 {
		t.Fatalf("matches = %v, want 1", done["matches"])
	}
	if done["saved_positions"].(float64) == 0 {
		t.Fatal("saved_positions = 0, want > 0")
	}
}

func TestImportUnsupportedFormat(t *testing.T) {
	ts := newTestServer(t)
	// imports.db is not wired yet → catch-all 404 (unknown route).
	resp := post(t, ts, "/v1/imports.db", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestImportCancelUnknownID(t *testing.T) {
	ts := newTestServer(t)
	resp := post(t, ts, "/v1/imports.cancel", importCancelReq{ImportID: "tenant-a-999"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestImportRegistryTenantScoping(t *testing.T) {
	reg := newImportRegistry()
	id := reg.start("tenant-a", func() {})
	// A different tenant cannot cancel tenant-a's import.
	if reg.cancel("tenant-b", id) {
		t.Fatal("tenant-b should not cancel tenant-a's import")
	}
	if !reg.cancel("tenant-a", id) {
		t.Fatal("tenant-a should cancel its own import")
	}
}
