package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// versionOnly is a storage.Storage whose only working method is Version —
// the one the health handlers call. Every other method is the nil embedded
// interface: reaching one is a bug in the handler, and shows up as a panic.
type versionOnly struct {
	storage.Storage
	version string
	err     error
}

func (v versionOnly) Version(context.Context) (string, error) { return v.version, v.err }

func get(t *testing.T, h http.HandlerFunc, path string) (int, map[string]string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, path, nil))
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q, want application/json", ct)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not a JSON object: %v\n%s", err, rec.Body.String())
	}
	return rec.Code, body
}

func TestReady_OK(t *testing.T) {
	h := &Health{Storage: versionOnly{version: "2.15.0"}, ExpectedVersion: "2.15.0"}
	code, body := get(t, h.Ready, "/readyz")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%v)", code, body)
	}
	if body["status"] != "ready" || body["version"] != "2.15.0" {
		t.Errorf("body = %v, want status=ready version=2.15.0", body)
	}
}

func TestReady_VersionMismatchIsNotReady(t *testing.T) {
	h := &Health{Storage: versionOnly{version: "2.14.0"}, ExpectedVersion: "2.15.0"}
	code, body := get(t, h.Ready, "/readyz")
	if code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (%v)", code, body)
	}
	if body["status"] != "version_mismatch" {
		t.Errorf("status field = %q, want version_mismatch", body["status"])
	}
	// Both versions are reported so an operator can tell a stale binary from
	// a database that has not been migrated.
	if body["version"] != "2.14.0" || body["expected"] != "2.15.0" {
		t.Errorf("body = %v, want version=2.14.0 expected=2.15.0", body)
	}
}

func TestReady_NoExpectedVersionAcceptsAny(t *testing.T) {
	h := &Health{Storage: versionOnly{version: "whatever"}}
	if code, _ := get(t, h.Ready, "/readyz"); code != http.StatusOK {
		t.Fatalf("status = %d, want 200 when ExpectedVersion is unset", code)
	}
}

func TestReadyAndLive_StorageDown(t *testing.T) {
	down := versionOnly{err: errors.New("connection refused")}
	for name, fn := range map[string]http.HandlerFunc{
		"live":  (&Health{Storage: down}).Live,
		"ready": (&Health{Storage: down, ExpectedVersion: "2.15.0"}).Ready,
	} {
		t.Run(name, func(t *testing.T) {
			code, body := get(t, fn, "/")
			if code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want 503", code)
			}
			if body["status"] != "down" {
				t.Errorf("status field = %q, want down", body["status"])
			}
			// The backend's error text must not leak into the probe answer.
			for k, v := range body {
				if v == "connection refused" {
					t.Errorf("field %q leaks the storage error", k)
				}
			}
		})
	}
}

func TestLive_OK(t *testing.T) {
	h := &Health{Storage: versionOnly{version: "1.0.0"}}
	code, body := get(t, h.Live, "/healthz")
	if code != http.StatusOK || body["status"] != "ok" {
		t.Fatalf("got %d %v, want 200 status=ok", code, body)
	}
}
