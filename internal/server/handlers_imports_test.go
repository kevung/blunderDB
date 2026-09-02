package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/internal/server/metrics"
	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
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
	if err := json.NewDecoder(saveResp.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
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

func TestImportNativeDBEndToEnd(t *testing.T) {
	ts := newTestServer(t)

	// Build a populated native .db by importing an XG fixture via the legacy
	// Database, then upload that .db to imports.db.
	dbPath := filepath.Join(t.TempDir(), "lib.db")
	db := database.NewDatabase()
	if err := db.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	if _, err := db.ImportXGMatch(filepath.Join("..", "..", "testdata", "match_with_comment.xg")); err != nil {
		db.Close()
		t.Fatalf("seed import: %v", err)
	}
	db.Close()

	blob, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("read .db: %v", err)
	}

	events := uploadImportNamed(t, ts, "/v1/imports.db", "lib.db", blob)
	done := events[len(events)-1]
	if done["event"] != "done" {
		t.Fatalf("last event = %v, want done", done["event"])
	}
	if done["saved_positions"].(float64) == 0 {
		t.Fatal("saved_positions = 0, want > 0")
	}
}

func TestImportXGPPositionEndToEnd(t *testing.T) {
	ts := newTestServer(t)

	fixture, err := os.ReadFile(filepath.Join("..", "..", "testdata", "xgp", "Position 10.xgp"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	events := uploadImportNamed(t, ts, "/v1/imports.position", "pos.xgp", fixture)
	done := events[len(events)-1]
	if done["event"] != "done" {
		t.Fatalf("last event = %v, want done", done["event"])
	}
	if done["saved_positions"].(float64) == 0 {
		t.Fatal("saved_positions = 0, want > 0")
	}
}

func TestImportBGFTextPositionEndToEnd(t *testing.T) {
	ts := newTestServer(t)

	fixture, err := os.ReadFile(filepath.Join("..", "..", "testdata", "bgf_positions", "01_checkerPosition_FR.txt"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	events := uploadImportNamed(t, ts, "/v1/imports.position", "pos.txt", fixture)
	done := events[len(events)-1]
	if done["event"] != "done" {
		t.Fatalf("last event = %v, want done", done["event"])
	}
	if done["saved_positions"].(float64) == 0 {
		t.Fatal("saved_positions = 0, want > 0")
	}
}

func TestImportUnsupportedFormat(t *testing.T) {
	ts := newTestServer(t)
	// An unknown imports.* verb hits the catch-all 404 (unknown route).
	resp := post(t, ts, "/v1/imports.bogus", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestImportCancelUnknownID(t *testing.T) {
	ts := newTestServer(t)
	resp := post(t, ts, "/v1/imports.cancel", importCancelReq{ImportID: "1-999"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestExportSQLiteSuccess(t *testing.T) {
	ts := newTestServer(t)

	p := domain.InitializePosition()
	post(t, ts, "/v1/positions.save", positionReq{Position: &p}).Body.Close()

	resp := post(t, ts, "/v1/exports.sqlite", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/octet-stream" {
		t.Fatalf("content-type = %q, want application/octet-stream", ct)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, "blunderdb-export.sqlite") {
		t.Fatalf("content-disposition = %q, want the .sqlite filename", cd)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(body, []byte("SQLite format 3")) {
		t.Fatalf("body does not look like a SQLite file (first bytes: %q)", body[:min(32, len(body))])
	}
}

// TestExportSQLiteIncludesMatches covers the fix for a known gap: the
// handler used to call Export with a zero-value ingest.Selection, so
// exports.sqlite carried nothing at all — no positions, and certainly no
// matches. It now runs ingest.WholeTenant, so a match imported through this
// same server must come back out.
func TestExportSQLiteIncludesMatches(t *testing.T) {
	ts := newTestServer(t)

	fixture, err := os.ReadFile(filepath.Join("..", "..", "testdata", "test.sgf"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	events := uploadImportNamed(t, ts, "/v1/imports.gnubg", "test.sgf", fixture)
	if done := events[len(events)-1]; done["matches"].(float64) != 1 {
		t.Fatalf("import matches = %v, want 1", done["matches"])
	}

	resp := post(t, ts, "/v1/exports.sqlite", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "export.db")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	db := database.NewDatabase()
	if err := db.OpenDatabase(path); err != nil {
		t.Fatalf("OpenDatabase(export): %v", err)
	}
	defer db.Close()

	matches, err := db.GetAllMatches()
	if err != nil {
		t.Fatalf("GetAllMatches: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("exported matches = %d, want 1", len(matches))
	}
	positions, err := db.LoadAllPositions()
	if err != nil {
		t.Fatalf("LoadAllPositions: %v", err)
	}
	if len(positions) == 0 {
		t.Fatal("exported positions = 0, want > 0")
	}
}

// TestExportSQLiteWatermarkRequiresIdentity covers the other known gap: a
// caller asking for a watermark on a server that was never given a signing
// identity (no --identity-dir) must get a clear error, not a silently
// unmarked file.
func TestExportSQLiteWatermarkRequiresIdentity(t *testing.T) {
	ts := newTestServer(t)

	resp := post(t, ts, "/v1/exports.sqlite", exportSQLiteReq{WatermarkOrigin: "cours du mardi"})
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatal("status = 200, want an error: this server has no identity to sign with")
	}
	var env errorEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("body not a JSON error envelope: %v", err)
	}
	if env.Error.Code != CodeInvalid {
		t.Fatalf("error code = %q, want %q", env.Error.Code, CodeInvalid)
	}
}

// TestExportSQLiteWatermark covers a daemon that does have a signing
// identity (Options.Identity, set by RunServe from --identity-dir): asking
// for a watermark on exports.sqlite must produce a file whose watermark
// verifies and carries the origin/note given, exactly like the desktop's
// export dialog.
func TestExportSQLiteWatermark(t *testing.T) {
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	identity, err := issuance.NewIdentity("blunderdb-serve-test")
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	srv, err := New(Options{Storage: st, Metrics: metrics.New(), Identity: identity})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	p := domain.InitializePosition()
	post(t, ts, "/v1/positions.save", positionReq{Position: &p}).Body.Close()

	resp := post(t, ts, "/v1/exports.sqlite", exportSQLiteReq{
		WatermarkOrigin: "cours du mardi",
		WatermarkNote:   "ne pas redistribuer",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200 (body=%s)", resp.StatusCode, body)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "export.db")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	db := database.NewDatabase()
	if err := db.OpenDatabase(path); err != nil {
		t.Fatalf("OpenDatabase(export): %v", err)
	}
	defer db.Close()

	info, err := db.GetIssuanceInfo()
	if err != nil {
		t.Fatalf("GetIssuanceInfo: %v", err)
	}
	if !info.Watermarked || info.Watermark == nil {
		t.Fatal("exported file is not watermarked")
	}
	if !info.Watermark.SignatureValid {
		t.Fatal("watermark signature does not verify")
	}
	if info.Watermark.Origin != "cours du mardi" {
		t.Fatalf("watermark origin = %q, want %q", info.Watermark.Origin, "cours du mardi")
	}
	if info.Watermark.Note != "ne pas redistribuer" {
		t.Fatalf("watermark note = %q, want %q", info.Watermark.Note, "ne pas redistribuer")
	}
}

// TestExportSQLiteErrorEnvelope covers the fix for the "200 on export
// failure" bug: an already-cancelled request context makes
// ingest.SQLiteExporter.Export fail (it opens its temp SQLite file with the
// same context), before the handler has written anything to the real
// ResponseWriter. The response must carry a proper error status and a JSON
// envelope — not HTTP 200 with Content-Type: application/octet-stream and a
// stray JSON body wearing a .sqlite Content-Disposition, which is what the
// handler used to produce.
func TestExportSQLiteErrorEnvelope(t *testing.T) {
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	srv, err := New(Options{Storage: st, Metrics: metrics.New()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodPost, "/v1/exports.sqlite", nil).WithContext(ctx)
	req.Header.Set(middleware.TenantHeader, testTenant)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf("status = %d, want a non-200 error status", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("content-type = %q, want application/json", ct)
	}
	if cd := rec.Header().Get("Content-Disposition"); cd != "" {
		t.Fatalf("content-disposition = %q, want empty on error", cd)
	}
	var env errorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("body not a JSON error envelope: %v (body=%s)", err, rec.Body.String())
	}
	if env.Error.Code == "" {
		t.Fatal("error envelope has an empty code")
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
