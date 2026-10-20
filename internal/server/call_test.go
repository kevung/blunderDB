package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestCallRouteCoverage asserts every route the dispatcher reaches is
// registered (never the catch-all 404); a 400 for an empty body is fine.
func TestCallRouteCoverage(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	srv, err := New(Options{Storage: s})
	if err != nil {
		t.Fatal(err)
	}

	paths := srv.Paths()
	if len(paths) < 100 {
		t.Fatalf("only %d domain routes; expected the full Storage surface (>=100)", len(paths))
	}
	for _, p := range paths {
		req := httptest.NewRequest(http.MethodPost, p, strings.NewReader("{}"))
		req.Header.Set(middleware.TenantHeader, "1")
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		// A 404 from a handler (ErrNotFound, e.g. load id 0) is fine; only the
		// catch-all "unknown route" envelope means the route is unregistered.
		if rec.Code == http.StatusNotFound && strings.Contains(rec.Body.String(), "unknown route") {
			t.Errorf("route %s hit the catch-all (not registered)", p)
		}
	}
}

// captureStdout runs fn with os.Stdout redirected to a pipe and returns what it
// wrote.
func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	// Drain concurrently: a Windows pipe buffer is smaller than --list output,
	// so reading only after fn returns deadlocks the writer.
	done := make(chan []byte)
	go func() {
		out, _ := io.ReadAll(r)
		done <- out
	}()
	os.Stdout = w
	runErr := fn()
	w.Close()
	os.Stdout = old
	out := <-done
	r.Close()
	return string(out), runErr
}

func TestRunCallList(t *testing.T) {
	out, err := captureStdout(t, func() error { return RunCall([]string{"--list"}) })
	if err != nil {
		t.Fatalf("RunCall --list: %v", err)
	}
	if !strings.Contains(out, "positions.save") || !strings.Contains(out, "metadata.counts") {
		t.Fatalf("--list missing expected methods:\n%s", out)
	}
	if n := len(strings.Fields(out)); n < 100 {
		t.Fatalf("--list printed %d methods, want >=100", n)
	}
}

func TestRunCallEndToEnd(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "call.db")

	pos := `{"position":{"board":{"points":[],"bearoff":[0,0]},"cube":{"owner":-1,"value":0},"dice":[3,1],"score":[7,7],"player_on_roll":0,"decision_type":0}}`
	out, err := captureStdout(t, func() error {
		return RunCall([]string{"positions.save", "--db", dbPath, "--json", pos})
	})
	if err != nil {
		t.Fatalf("positions.save: %v (out=%s)", err, out)
	}
	var saved struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal([]byte(out), &saved); err != nil {
		t.Fatalf("save response not JSON: %q", out)
	}
	if saved.ID == 0 {
		t.Fatalf("save returned id 0: %s", out)
	}

	out, err = captureStdout(t, func() error {
		return RunCall([]string{"metadata.counts", "--db", dbPath})
	})
	if err != nil {
		t.Fatalf("metadata.counts: %v", err)
	}
	var counts struct {
		Positions int `json:"positions"`
	}
	if err := json.Unmarshal([]byte(out), &counts); err != nil {
		t.Fatalf("counts response not JSON: %q", out)
	}
	if counts.Positions != 1 {
		t.Fatalf("positions = %d, want 1", counts.Positions)
	}
}

// TestStdoutResponseWriter_ImplicitStatus: without WriteHeader the status is
// 200, and bytes reach stdout on Write.
func TestStdoutResponseWriter_ImplicitStatus(t *testing.T) {
	out, err := captureStdout(t, func() error {
		w := newStdoutResponseWriter()
		if _, werr := w.Write([]byte("hello")); werr != nil {
			return werr
		}
		if w.status != http.StatusOK {
			t.Errorf("status after an unheadered Write = %d, want %d", w.status, http.StatusOK)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "hello" {
		t.Errorf("stdout = %q, want %q", out, "hello")
	}
}

// TestStdoutResponseWriter_ExplicitStatus: an explicit status is kept, and a
// second WriteHeader does not override it (net/http's contract).
func TestStdoutResponseWriter_ExplicitStatus(t *testing.T) {
	w := newStdoutResponseWriter()
	w.WriteHeader(http.StatusNotFound)
	w.WriteHeader(http.StatusInternalServerError) // must be ignored
	if w.status != http.StatusNotFound {
		t.Errorf("status = %d, want %d (first WriteHeader wins)", w.status, http.StatusNotFound)
	}
}

// TestRunCallErrorExit verifies an unknown method prints the error envelope and
// returns a non-zero (error) result.
func TestRunCallErrorExit(t *testing.T) {
	out, err := captureStdout(t, func() error {
		return RunCall([]string{"positions.bogus", "--db", filepath.Join(t.TempDir(), "x.db")})
	})
	if err == nil {
		t.Fatal("expected an error for an unknown method")
	}
	if !strings.Contains(out, `"error"`) {
		t.Fatalf("expected an error envelope on stdout, got: %q", out)
	}
}

// TestRunCallRejectsNamedScope: `call --scope alice` fails before anything is
// opened, naming the flag and the expected format (ADR-0005).
func TestRunCallRejectsNamedScope(t *testing.T) {
	for _, scope := range []string{"alice", "default", "0"} {
		err := RunCall([]string{"metadata.counts", "--db", "/nonexistent/x.db", "--scope", scope})
		if err == nil {
			t.Fatalf("--scope %q accepted", scope)
		}
		for _, want := range []string{"--scope", storage.TenantFormat, `"` + scope + `"`} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("--scope %q: error %q does not mention %q", scope, err, want)
			}
		}
	}
}

// TestCallRemoteBackend runs `call` over a daemon's /v1: the in-process
// handlers reach the remote daemon's database through storage/remote, with
// the configured header on every request.
func TestCallRemoteBackend(t *testing.T) {
	ts, _ := newTestServerAndHandler(t)
	var token string
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token = r.Header.Get("X-Proxy-Token")
		ts.Config.Handler.ServeHTTP(w, r)
	}))
	t.Cleanup(proxy.Close)

	if _, err := captureStdout(t, func() error {
		return RunCall([]string{"positions.save", "--backend", "remote", "--dsn", proxy.URL,
			"--header", "X-Proxy-Token: s3cret", "--json", `{"position":{"decision_type":0}}`})
	}); err != nil {
		t.Fatalf("positions.save over remote: %v", err)
	}
	if token != "s3cret" {
		t.Fatalf("--header not forwarded: got %q", token)
	}
	out, err := captureStdout(t, func() error {
		return RunCall([]string{"positions.count", "--backend", "remote", "--dsn", proxy.URL})
	})
	if err != nil || strings.TrimSpace(out) != "1" {
		t.Fatalf("positions.count over remote = %q, %v; want 1", out, err)
	}
}

// TestCallRemoteNamesTheUnsupportedMethod: a route whose storage method /v1
// does not serve answers an internal error, but one that names the method,
// not the generic text that hides a backend's internals.
func TestCallRemoteNamesTheUnsupportedMethod(t *testing.T) {
	ts, _ := newTestServerAndHandler(t)
	out, err := captureStdout(t, func() error {
		return RunCall([]string{"met.list", "--backend", "remote", "--dsn", ts.URL})
	})
	if err == nil {
		t.Fatalf("met.list over remote succeeded: %s", out)
	}
	if !strings.Contains(out, `"internal"`) || !strings.Contains(out, "MatchEquityTableStore.") || !strings.Contains(out, "is not served through the remote backend") {
		t.Fatalf("envelope does not name the method: %s", out)
	}
}

func TestCallRemoteRefusesDbAndTenantHeader(t *testing.T) {
	if err := RunCall([]string{"metadata.counts", "--backend", "remote", "--dsn", "http://x", "--db", "a.db"}); err == nil || !strings.Contains(err.Error(), "exclude each other") {
		t.Fatalf("--db with remote: %v", err)
	}
	if err := RunCall([]string{"metadata.counts", "--backend", "remote", "--dsn", "http://x", "--header", "X-Tenant-ID: 9"}); err == nil || !strings.Contains(err.Error(), "--scope") {
		t.Fatalf("X-Tenant-ID header: %v", err)
	}
	if err := RunCall([]string{"metadata.counts", "--backend", "remote"}); err == nil || !strings.Contains(err.Error(), "--dsn") {
		t.Fatalf("missing dsn: %v", err)
	}
}

func TestCallRemoteProxyRefusalAndTimeout(t *testing.T) {
	deny := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "<html>sign in</html>", http.StatusUnauthorized)
	}))
	t.Cleanup(deny.Close)
	err := RunCall([]string{"metadata.counts", "--backend", "remote", "--dsn", deny.URL})
	if err == nil || !strings.Contains(err.Error(), "refused") || !strings.Contains(err.Error(), "--header") {
		t.Fatalf("401 reads as %v", err)
	}
	stall := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(stall.Close)
	start := time.Now()
	err = RunCall([]string{"metadata.counts", "--backend", "remote", "--dsn", stall.URL, "--timeout", "300ms"})
	if err == nil || time.Since(start) > 5*time.Second {
		t.Fatalf("stalled daemon: %v after %v", err, time.Since(start))
	}
}

// A stream that fails after its 200 still ends the process non-zero.
func TestStdoutResponseWriterErrorLineFails(t *testing.T) {
	w := newStdoutResponseWriter()
	_, _ = w.Write([]byte("{\"id\":1}\n{\"error\":{\"code\":\"internal\"}}\n"))
	if w.failure == "" {
		t.Fatal("error line not seen")
	}
}
