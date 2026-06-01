package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/internal/server/metrics"
	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// newTestServer builds a Server backed by a fresh in-memory SQLite database.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	srv, err := New(Options{
		Storage:       st,
		Metrics:       metrics.New(),
		EnableMetrics: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func TestHealthz(t *testing.T) {
	ts := newTestServer(t)
	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status field = %q, want ok", body["status"])
	}
}

func TestReadyzVersionMatches(t *testing.T) {
	ts := newTestServer(t)
	resp, err := http.Get(ts.URL + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["version"] != domain.DatabaseVersion {
		t.Fatalf("version = %q, want %q", body["version"], domain.DatabaseVersion)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	ts := newTestServer(t)
	// Generate one request so a counter exists.
	if _, err := http.Get(ts.URL + "/healthz"); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(ts.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	out, _ := io.ReadAll(resp.Body)
	text := string(out)
	if !strings.Contains(text, "blunderdb_http_requests_total") {
		t.Fatalf("metrics output missing request counter:\n%s", text)
	}
	if !strings.Contains(text, "blunderdb_http_request_duration_seconds_bucket") {
		t.Fatalf("metrics output missing latency histogram:\n%s", text)
	}
}

func TestTenantHeaderRequiredOnV1(t *testing.T) {
	ts := newTestServer(t)
	// /v1 paths require a tenant; without it → 400 invalid envelope.
	resp, err := http.Post(ts.URL+"/v1/positions.list", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var env errorEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != CodeInvalid {
		t.Fatalf("code = %q, want %q", env.Error.Code, CodeInvalid)
	}
}

func TestTenantHeaderReachesCatchAll(t *testing.T) {
	ts := newTestServer(t)
	// With a tenant header, the request passes the tenant gate and reaches the
	// catch-all (no /v1 routes yet in PR1) → 404 not_found envelope.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/positions.list", strings.NewReader("{}"))
	req.Header.Set(middleware.TenantHeader, "tenant-a")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	var env errorEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != CodeNotFound {
		t.Fatalf("code = %q, want %q", env.Error.Code, CodeNotFound)
	}
}

func TestUnknownRouteEnvelope(t *testing.T) {
	ts := newTestServer(t)
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/nope", nil)
	req.Header.Set(middleware.TenantHeader, "tenant-a")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("content-type = %q, want application/json", ct)
	}
}
