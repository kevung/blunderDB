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

	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestCallRouteCoverage asserts every domain route the dispatcher can reach is
// actually registered (never the catch-all 404). Posting an empty body may
// yield a 400 error envelope for methods that need fields — that is fine; what
// matters is that the route exists and the handler runs.
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
		req.Header.Set(middleware.TenantHeader, "t")
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
	os.Stdout = w
	runErr := fn()
	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
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
