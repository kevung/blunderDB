//go:build postgres

package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestMaintenanceVacuumPostgresNotSupported: the PostgreSQL-backed server has
// no file to compact and must refuse with CodeInvalid rather than 404 or
// pretend. Needs Docker (testcontainers), like TestTenantPurgeEndpoint.
func TestMaintenanceVacuumPostgresNotSupported(t *testing.T) {
	ts := newPostgresTestServer(t)

	resp := post(t, ts, "/ops/maintenance.vacuum", nil)
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
	if !strings.Contains(env.Error.Message, "not supported") {
		t.Fatalf("message = %q, want it to mention %q", env.Error.Message, "not supported")
	}
}
