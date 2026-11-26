package server

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The daemon writes a paper archive and reads it back: exports.archive serves
// the page, imports.archive takes it as an upload into another library and
// finds the match already there on a second pass.
func TestArchiveExportImportRoundTrip(t *testing.T) {
	src := newTestServer(t)
	mat, err := os.ReadFile(filepath.Join("..", "..", "testdata", "charlot1-charlot2_7p_2025-11-08-2305.mat"))
	if err != nil {
		t.Fatal(err)
	}
	if ev := uploadImportNamed(t, src, "/v1/imports.gnubg", "m.mat", mat); ev[len(ev)-1]["event"] != "done" {
		t.Fatalf("seed import: %v", ev[len(ev)-1])
	}

	resp := post(t, src, "/v1/exports.archive", map[string]any{"origin": "Club du démon"})
	page, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("exports.archive: status %d, %.200q", resp.StatusCode, page)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type = %q", ct)
	}
	var rep struct{ Matches, Codes int }
	if err := json.Unmarshal([]byte(resp.Header.Get(ExportReportHeader)), &rep); err != nil || rep.Matches != 1 || rep.Codes == 0 {
		t.Fatalf("report %+v (%v)", rep, err)
	}
	if !strings.Contains(string(page), "Club du démon") {
		t.Fatal("origin line missing from the page")
	}

	dst := newTestServer(t)
	done := uploadImportNamed(t, dst, "/v1/imports.archive", "archive.html", page)
	last := done[len(done)-1]
	if last["event"] != "done" || last["matches"].(float64) != 1 {
		t.Fatalf("archive import: %v", last)
	}
	again := uploadImportNamed(t, dst, "/v1/imports.archive", "archive.html", page)
	if last := again[len(again)-1]; last["event"] != "done" || last["matches"].(float64) != 0 {
		t.Fatalf("second archive import: %v", last)
	}

	// An empty tenant has nothing to archive: a refusal, not an empty page.
	resp = post(t, newTestServer(t), "/v1/exports.archive", nil)
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatal("exports.archive on an empty tenant answered 200")
	}
}
