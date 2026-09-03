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
