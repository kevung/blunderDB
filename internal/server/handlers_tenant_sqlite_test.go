// TestTenantPurgeSQLiteNotSupported never touches Postgres or
// testcontainers-go, unlike TestTenantPurgeEndpoint (handlers_tenant_test.go,
// tagged `//go:build postgres`). It lives in its own untagged file so it
// runs on the default `go test ./...` path (no Docker required), instead of
// only under `-tags postgres`.
package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestTenantPurgeSQLiteNotSupported confirms the endpoint refuses to run
// against a SQLite-backed server: SQLite has no tenant concept to purge, so
// the handler must 400 with CodeInvalid rather than silently no-op — and the
// (untouched) SQLite data must still be there afterwards.
func TestTenantPurgeSQLiteNotSupported(t *testing.T) {
	ts := newTestServer(t)

	p := domain.InitializePosition()
	saveResp := post(t, ts, "/v1/positions.save", positionReq{Position: &p})
	defer saveResp.Body.Close()
	var saved idResp
	if err := json.NewDecoder(saveResp.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}

	purgeResp := post(t, ts, "/v1/tenant.purge", nil)
	defer purgeResp.Body.Close()
	if purgeResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", purgeResp.StatusCode)
	}
	var env errorEnvelope
	if err := json.NewDecoder(purgeResp.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != CodeInvalid {
		t.Fatalf("code = %q, want %q", env.Error.Code, CodeInvalid)
	}
	if !strings.Contains(env.Error.Message, "not supported") {
		t.Fatalf("message = %q, want it to mention %q", env.Error.Message, "not supported")
	}

	// The SQLite data must be untouched by the rejected purge attempt.
	loadResp := post(t, ts, "/v1/positions.load", idReq{ID: saved.ID})
	defer loadResp.Body.Close()
	if loadResp.StatusCode != http.StatusOK {
		t.Fatalf("post-attempted-purge load status = %d, want 200 (SQLite data must be untouched)", loadResp.StatusCode)
	}
}
