package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestProbeURL(t *testing.T) {
	for in, want := range map[string]string{
		":8080":                  "http://127.0.0.1:8080",
		"0.0.0.0:8080":           "http://127.0.0.1:8080",
		"[::]:8080":              "http://[::1]:8080",
		"127.0.0.1:9000":         "http://127.0.0.1:9000",
		"localhost:9000":         "http://localhost:9000",
		"[::1]:9000":             "http://[::1]:9000",
		"":                       "http://127.0.0.1:8080",
		"localhost":              "http://localhost:8080",
		"http://proxy:8080/":     "http://proxy:8080",
		"https://blunderdb.test": "https://blunderdb.test",
	} {
		if got := ProbeURL(in); got != want {
			t.Errorf("ProbeURL(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestHealthcheck_Ready is the happy path against a real daemon: the probe
// speaks to the same /readyz the routing table serves.
func TestHealthcheck_Ready(t *testing.T) {
	ts := newTestServer(t)
	status, err := Healthcheck(context.Background(), strings.TrimPrefix(ts.URL, "http://"))
	if err != nil {
		t.Fatalf("Healthcheck: %v", err)
	}
	if status != "ready" {
		t.Errorf("status = %q, want ready", status)
	}
}

// TestHealthcheck_NotReady: a 503 is a failure that names the daemon's
// reason, so `docker inspect` shows "version_mismatch" rather than a bare
// exit code.
func TestHealthcheck_NotReady(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/readyz" {
			t.Errorf("probed %s, want /readyz", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"version_mismatch","version":"2.14.0","expected":"2.15.0"}`))
	}))
	defer ts.Close()
	_, err := Healthcheck(context.Background(), ts.URL)
	if err == nil {
		t.Fatal("Healthcheck succeeded on a 503")
	}
	if !strings.Contains(err.Error(), "version_mismatch") {
		t.Errorf("error %q does not carry the daemon's status word", err)
	}
}

// TestHealthcheck_ForeignServer: a 200 from something that is not a
// blunderdb daemon (a placeholder page squatting the port) is not "ready".
func TestHealthcheck_ForeignServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html>It works!</html>"))
	}))
	defer ts.Close()
	if _, err := Healthcheck(context.Background(), ts.URL); err == nil {
		t.Fatal("Healthcheck accepted a non-JSON 200")
	}
}

// TestHealthcheck_Unreachable: nothing listening is the commonest failure
// (daemon still starting, or crashed); it must be an error, not a hang.
func TestHealthcheck_Unreachable(t *testing.T) {
	ts := httptest.NewServer(http.NotFoundHandler())
	addr := ts.Listener.Addr().String()
	ts.Close() // the port is now free: connection refused
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := Healthcheck(ctx, addr); err == nil {
		t.Fatal("Healthcheck succeeded with nothing listening")
	}
}

// TestRunHealthcheck_Flags: --addr overrides the environment, and the exit
// status the CLI maps the error to is what a HEALTHCHECK reads.
func TestRunHealthcheck_Flags(t *testing.T) {
	ts := newTestServer(t)
	t.Setenv("BLUNDERDB_ADDR", "127.0.0.1:1") // nothing listens there
	if err := RunHealthcheck(nil); err == nil {
		t.Fatal("RunHealthcheck(env → unreachable) succeeded")
	}
	if err := RunHealthcheck([]string{"--addr", strings.TrimPrefix(ts.URL, "http://")}); err != nil {
		t.Fatalf("RunHealthcheck(--addr) = %v", err)
	}
	if err := RunHealthcheck([]string{"--bogus"}); err == nil {
		t.Fatal("unknown flag accepted")
	}
}
