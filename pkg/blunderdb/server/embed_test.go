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
// unchanged) with a 400 the embedder can parse. The route under test used to
// be the singular, non-existent "/v1/position.save": every request landed on
// the catch-all mux and got its generic 404, so the assertion below
// ("not 200") passed without ever exercising the embedding's actual
// wiring — see issue #220.
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

// TestBootstrapAppliesCORSAllowOrigin guards #236: Config.CORSAllowOrigin
// used to have no way to reach internal/server.Options at all — an embedder
// could never turn CORS on for its own front end. A request with a matching
// Origin header must now get it echoed back.
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

// TestBootstrapAppliesMaxBodyBytes guards #236: Config.MaxBodyBytes used to
// have no way to reach internal/server.Options — every embedder was stuck
// with the internal daemon's own default cap, unable to tighten (or loosen)
// it for its own deployment. A body over the configured cap must be refused
// with 413, declared Content-Length or not.
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
