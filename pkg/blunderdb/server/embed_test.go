package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Bootstrap returns a working engine handler over an in-memory SQLite store:
// a /healthz probe answers 2xx without an X-Tenant-ID header (public path).
func TestBootstrapServesHealthz(t *testing.T) {
	h, closer, err := Bootstrap(context.Background(), Config{
		Backend: "sqlite", DSN: ":memory:", EnableMetrics: true,
	})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	defer closer.Close()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	h.ServeHTTP(rec, req)
	if rec.Code/100 != 2 {
		t.Fatalf("healthz: got %d, want 2xx (body %q)", rec.Code, rec.Body.String())
	}

	// A /v1 call still requires the tenant header (middleware unchanged).
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/v1/position.save", bytes.NewReader([]byte("{}")))
	h.ServeHTTP(rec2, req2)
	if rec2.Code == http.StatusOK {
		t.Fatalf("/v1 without X-Tenant-ID should not be 200")
	}
}
