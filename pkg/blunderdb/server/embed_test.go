package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/internal/server/middleware"
)

// errorEnvelope mirrors the wire shape of internal/server's error responses
// ({"error":{"code":"...","message":"..."}}) — that type is unexported, so an
// embedder (this package, and any real one like gammonGo) can only observe it
// over the wire, exactly like this test does.
type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// Bootstrap returns a working engine handler over an in-memory SQLite store:
// a /healthz probe answers 2xx without an X-Tenant-ID header (public path),
// and a real /v1 route still enforces the tenant header (middleware
// unchanged) with a 400 the embedder can parse. The route must be a real one:
// an unknown route hits the catch-all 404 and would pass "not 200" vacuously.
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

	// A real /v1 route still requires the tenant header (middleware unchanged).
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/v1/positions.save", bytes.NewReader([]byte("{}")))
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("/v1/positions.save without %s: status %d, want %d (body %q)",
			middleware.TenantHeader, rec2.Code, http.StatusBadRequest, rec2.Body.String())
	}
	var env errorEnvelope
	if err := json.NewDecoder(rec2.Body).Decode(&env); err != nil {
		t.Fatalf("decode error envelope: %v (body %q)", err, rec2.Body.String())
	}
	if env.Error.Code != "invalid" {
		t.Errorf("code = %q, want %q", env.Error.Code, "invalid")
	}
	if !strings.Contains(env.Error.Message, middleware.TenantHeader) {
		t.Errorf("message = %q, want it to name %q", env.Error.Message, middleware.TenantHeader)
	}
}

// TestBootstrapAppliesCORSAllowOrigin: Config.CORSAllowOrigin reaches
// internal/server.Options — a matching Origin header is echoed back.
func TestBootstrapAppliesCORSAllowOrigin(t *testing.T) {
	h, closer, err := Bootstrap(context.Background(), Config{
		Backend:         "sqlite",
		DSN:             ":memory:",
		CORSAllowOrigin: "https://example.test",
	})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	defer closer.Close()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("Origin", "https://example.test")
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://example.test" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "https://example.test")
	}
}

// TestBootstrapAppliesMaxBodyBytes: Config.MaxBodyBytes reaches
// internal/server.Options — a body over the cap is refused with 413,
// declared Content-Length or not.
func TestBootstrapAppliesMaxBodyBytes(t *testing.T) {
	h, closer, err := Bootstrap(context.Background(), Config{
		Backend:      "sqlite",
		DSN:          ":memory:",
		MaxBodyBytes: 16,
	})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	defer closer.Close()

	body := bytes.NewReader([]byte(`{"zobrist": 1, "padding": "this is well over sixteen bytes"}`))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/positions.exists", body)
	req.Header.Set(middleware.TenantHeader, "1")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want %d (body %q)", rec.Code, http.StatusRequestEntityTooLarge, rec.Body.String())
	}
}
