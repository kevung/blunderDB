package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/internal/server/metrics"
	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// The three holes of #160: raw err.Error() on the wire, a body cap exempting
// a whole prefix, and a prefix-matched import registry. Each test below is
// the recette of one "À faire" line of task A.6.

// secretCause is what a backend error looks like: it quotes a DSN, a path,
// a statement. None of it may reach a client.
const secretCause = "pq: connect postgres://app:hunter2@db.internal/blunderdb: relation \"positions\" does not exist"

// failingOps wraps a Storage so that the duck-typed maintenance methods
// (vacuumer, tenantPurger) fail with a non-sentinel — hence internal — error.
type failingOps struct{ storage.Storage }

func (failingOps) Vacuum(context.Context) (storage.VacuumResult, error) {
	return storage.VacuumResult{}, errors.New(secretCause)
}

func (failingOps) PurgeTenant(context.Context, string) error {
	return errors.New(secretCause)
}

// newLoggedServer builds a Server on a wrapped in-memory store, with a
// logger captured into buf, so a test can assert both sides of the masking:
// the generic body the client sees, the real cause in the server log.
func newLoggedServer(t *testing.T, wrap func(storage.Storage) storage.Storage) (*Server, *strings.Builder) {
	t.Helper()
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	var store storage.Storage = st
	if wrap != nil {
		store = wrap(st)
	}
	var buf strings.Builder
	srv, err := New(Options{
		Storage: store,
		Metrics: metrics.New(),
		Logger:  slog.New(slog.NewTextHandler(&buf, nil)),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return srv, &buf
}

// serve runs one request through the full middleware chain as tenant and
// returns the recorder.
func serve(t *testing.T, srv *Server, ctx context.Context, tenant, path string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, body).WithContext(ctx)
	req.Header.Set(middleware.TenantHeader, tenant)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

// assertMasked checks the two halves of the contract on a response body
// (an envelope, or one NDJSON line/event carrying an "error" object): the
// client sees exactly {"code":"internal","message":"internal error"} and
// nothing of the cause; the server log carries the cause at Error level.
func assertMasked(t *testing.T, what string, body errorBody, raw string, log string) {
	t.Helper()
	want := errorBody{Code: CodeInternal, Message: "internal error"}
	if body.Code != want.Code || body.Message != want.Message || body.Details != nil {
		t.Errorf("%s: error body = %+v, want %+v", what, body, want)
	}
	for _, leak := range []string{"hunter2", "postgres://", "relation", "context canceled"} {
		if strings.Contains(raw, leak) {
			t.Errorf("%s: client body leaks %q:\n%s", what, leak, raw)
		}
	}
	if !strings.Contains(log, "level=ERROR") {
		t.Errorf("%s: server log not at Error level:\n%s", what, log)
	}
}

func TestInternalErrorMasked_MaintenanceVacuum(t *testing.T) {
	srv, log := newLoggedServer(t, func(s storage.Storage) storage.Storage { return failingOps{s} })
	rec := serve(t, srv, context.Background(), testTenant, "/v1/maintenance.vacuum", nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (%s)", rec.Code, rec.Body)
	}
	var env errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	assertMasked(t, "maintenance.vacuum", env.Error, rec.Body.String(), log.String())
	if !strings.Contains(log.String(), "hunter2") {
		t.Errorf("server log lost the real cause:\n%s", log.String())
	}
}

func TestInternalErrorMasked_TenantPurge(t *testing.T) {
	srv, log := newLoggedServer(t, func(s storage.Storage) storage.Storage { return failingOps{s} })
	rec := serve(t, srv, context.Background(), testTenant, "/v1/tenant.purge", nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (%s)", rec.Code, rec.Body)
	}
	var env errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	assertMasked(t, "tenant.purge", env.Error, rec.Body.String(), log.String())
	if !strings.Contains(log.String(), "hunter2") {
		t.Errorf("server log lost the real cause:\n%s", log.String())
	}
}

// TestInternalErrorMasked_MatchesExportMat: exportMatchMATHandler used to
// write codeForErr(err) with err.Error() — right code, raw message. A
// cancelled request context makes the storage read fail with a plain error.
func TestInternalErrorMasked_MatchesExportMat(t *testing.T) {
	srv, log := newLoggedServer(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rec := serve(t, srv, ctx, testTenant, "/v1/matches.exportMat", strings.NewReader(`{"matchId":1}`))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (%s)", rec.Code, rec.Body)
	}
	var env errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	assertMasked(t, "matches.exportMat", env.Error, rec.Body.String(), log.String())
}

// lastErrorObject returns the "error" object of the last NDJSON line of an
// event stream (handleImport / gammonnet emit {"event":"error","error":{…}};
// streamSeq2 / handleExport emit an envelope {"error":{…}}).
func lastErrorObject(t *testing.T, raw []byte) errorBody {
	t.Helper()
	lines := bytes.Split(bytes.TrimSpace(raw), []byte("\n"))
	var last struct {
		Error errorBody `json:"error"`
	}
	if err := json.Unmarshal(lines[len(lines)-1], &last); err != nil {
		t.Fatalf("last line not JSON: %q", lines[len(lines)-1])
	}
	return last.Error
}

// TestInternalErrorMasked_NDJSONStreams: the four sites that have already
// committed a 200 and can only append a trailing error — the streamed rows
// (rpcStream/streamSeq2), an import's error event, an export's trailing
// envelope, and the gammonNet sweep's error event — mask the same way.
func TestInternalErrorMasked_NDJSONStreams(t *testing.T) {
	cases := []struct {
		name, path  string
		body        func() (io.Reader, string)
		wantEvent   string
		streamStart bool
	}{
		{"positions.list", "/v1/positions.list", func() (io.Reader, string) { return strings.NewReader(`{}`), "application/json" }, "", false},
		{"exports.json", "/v1/exports.json", func() (io.Reader, string) { return nil, "application/json" }, "", false},
		{"gammonnet.analyzeMissing", "/v1/gammonnet.analyzeMissing", func() (io.Reader, string) { return nil, "application/json" }, "error", true},
		{"imports.json", "/v1/imports.json", func() (io.Reader, string) {
			var b bytes.Buffer
			mw := newMultipart(t, &b, "data.ndjson", []byte(`{"kind":"position"}`+"\n"))
			return &b, mw
		}, "error", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, log := newLoggedServer(t, nil)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			body, ct := tc.body()
			req := httptest.NewRequest(http.MethodPost, tc.path, body).WithContext(ctx)
			req.Header.Set(middleware.TenantHeader, testTenant)
			req.Header.Set("Content-Type", ct)
			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, req)

			raw := rec.Body.String()
			var got errorBody
			if tc.streamStart {
				// Handlers that emit "started" first are committed to 200.
				if rec.Code != http.StatusOK {
					t.Fatalf("status = %d, want 200 (%s)", rec.Code, raw)
				}
				var ev map[string]any
				lines := strings.Split(strings.TrimSpace(raw), "\n")
				if err := json.Unmarshal([]byte(lines[len(lines)-1]), &ev); err != nil {
					t.Fatal(err)
				}
				if ev["event"] != tc.wantEvent {
					t.Fatalf("last event = %v, want %q (%s)", ev["event"], tc.wantEvent, raw)
				}
				got = lastErrorObject(t, rec.Body.Bytes())
			} else {
				var env errorEnvelope
				if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
					t.Fatalf("body not an envelope: %s", raw)
				}
				got = env.Error
			}
			assertMasked(t, tc.name, got, raw, log.String())
			if !strings.Contains(log.String(), "context canceled") {
				t.Errorf("server log lost the real cause:\n%s", log.String())
			}
		})
	}
}

// newMultipart writes a one-file multipart body to w and returns its
// Content-Type.
func newMultipart(t *testing.T, w io.Writer, filename string, file []byte) string {
	t.Helper()
	mw := multipart.NewWriter(w)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(file); err != nil {
		t.Fatal(err)
	}
	mw.Close()
	return mw.FormDataContentType()
}

// --- body cap ---------------------------------------------------------

// openString yields a JSON string that never closes — `"` then n-1 bytes
// of 'a' — so a decoder keeps reading, with nothing to reject, until the
// cap cuts it off.
func openString(n int64) io.Reader {
	chunk := bytes.Repeat([]byte("a"), 1<<16)
	rs := []io.Reader{strings.NewReader(`"`)}
	for left := n - 1; left > 0; left -= 1 << 16 {
		rs = append(rs, io.LimitReader(bytes.NewReader(chunk), min(left, 1<<16)))
	}
	return io.MultiReader(rs...)
}

// TestImportsCancel_BodyTooLarge: imports.cancel is not an upload, so a 64 MiB
// body hits the default cap (32 MiB) — declared up front (Content-Length) or
// streamed with no length, the answer is 413.
func TestImportsCancel_BodyTooLarge(t *testing.T) {
	const size = 64 << 20
	for _, declared := range []bool{true, false} {
		name := "chunked"
		if declared {
			name = "content-length"
		}
		t.Run(name, func(t *testing.T) {
			srv, _ := newLoggedServer(t, nil)
			req := httptest.NewRequest(http.MethodPost, "/v1/imports.cancel", openString(size))
			req.ContentLength = -1
			if declared {
				req.ContentLength = size
			}
			req.Header.Set(middleware.TenantHeader, testTenant)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, req)
			if rec.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("status = %d, want 413 (%s)", rec.Code, rec.Body)
			}
			var env errorEnvelope
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
				t.Fatal(err)
			}
			if env.Error.Code != CodeInvalid {
				t.Errorf("code = %q, want %q", env.Error.Code, CodeInvalid)
			}
		})
	}
}

// TestLimitBody_ExemptsExactlyTheUploads: the exemption is the six upload
// routes, not the "/v1/imports." prefix. A 64 MiB declared body on an
// upload route gets past limitBody (and fails later, for not being
// multipart — 400, not 413).
func TestLimitBody_ExemptsExactlyTheUploads(t *testing.T) {
	want := map[string]bool{
		"/v1/imports.json": true, "/v1/imports.xg": true, "/v1/imports.gnubg": true,
		"/v1/imports.bgf": true, "/v1/imports.db": true, "/v1/imports.position": true,
	}
	got := uploadPaths()
	if len(got) != len(want) {
		t.Fatalf("uploadPaths = %v, want %v", got, want)
	}
	for p := range want {
		if !got[p] {
			t.Errorf("uploadPaths missing %s", p)
		}
	}
	if got["/v1/imports.cancel"] {
		t.Error("imports.cancel must not be exempt from the body cap")
	}

	srv, _ := newLoggedServer(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/imports.json", strings.NewReader("{}"))
	req.ContentLength = 64 << 20
	req.Header.Set(middleware.TenantHeader, testTenant)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("upload route with a 64 MiB Content-Length: status = %d, want 400 (not multipart), body %s", rec.Code, rec.Body)
	}
}

// --- import registry --------------------------------------------------

var hex32 = regexp.MustCompile(`^[0-9a-f]{32}$`)

func TestImportRegistry_OpaqueIDs(t *testing.T) {
	reg := newImportRegistry()
	a := reg.start("tenant-a", func() {})
	b := reg.start("tenant-a", func() {})
	for _, id := range []string{a, b} {
		if !hex32.MatchString(id) {
			t.Errorf("id %q is not 16 random bytes in hex", id)
		}
		if strings.Contains(id, "tenant") {
			t.Errorf("id %q carries the scope", id)
		}
	}
	if a == b {
		t.Errorf("two imports got the same id %q", a)
	}
}

// TestImportRegistry_ScopeByEquality: the owning tenant is compared by
// equality, so neither the prefix pair of the report ("a" vs "a-1") nor the
// numeric pair of the recette ("1" vs "10") can cancel each other's import.
func TestImportRegistry_ScopeByEquality(t *testing.T) {
	for _, pair := range [][2]string{{"a", "a-1"}, {"1", "10"}, {"10", "1"}} {
		attacker, owner := pair[0], pair[1]
		reg := newImportRegistry()
		cancelled := false
		id := reg.start(owner, func() { cancelled = true })
		if reg.cancel(attacker, id) {
			t.Errorf("tenant %q cancelled the import of tenant %q", attacker, owner)
		}
		if cancelled {
			t.Errorf("tenant %q's import was aborted by tenant %q", owner, attacker)
		}
		if !reg.cancel(owner, id) || !cancelled {
			t.Errorf("tenant %q could not cancel its own import", owner)
		}
	}
}

// TestImportCancel_CrossTenantHTTP is the same guarantee over the wire:
// tenant "1" asking imports.cancel for tenant "10"'s id gets the same 404 an
// unknown id gets — it learns nothing — and tenant "10" gets its {ok:true}.
func TestImportCancel_CrossTenantHTTP(t *testing.T) {
	srv, _ := newLoggedServer(t, nil)
	cancelled := false
	id := srv.imports.start("10", func() { cancelled = true })
	t.Cleanup(func() { srv.imports.finish(id) })
	body := func() io.Reader { return strings.NewReader(`{"importId":"` + id + `"}`) }

	rec := serve(t, srv, context.Background(), "1", "/v1/imports.cancel", body())
	if rec.Code != http.StatusNotFound || cancelled {
		t.Fatalf("tenant 1 cancelling tenant 10's import: status = %d, cancelled = %v; want 404, false", rec.Code, cancelled)
	}
	rec = serve(t, srv, context.Background(), "10", "/v1/imports.cancel", body())
	if rec.Code != http.StatusOK || !cancelled {
		t.Fatalf("tenant 10 cancelling its own import: status = %d, cancelled = %v; want 200, true", rec.Code, cancelled)
	}
}
