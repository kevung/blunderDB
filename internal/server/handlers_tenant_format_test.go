// The tenant-format tests below run against the SQLite-backed test server,
// so they need neither PostgreSQL nor Docker and run on the default
// `go test ./...` path. The PostgreSQL counterpart, TestTenantIsolationNamed
// (handlers_tenant_test.go, tagged `//go:build postgres`), covers the part
// SQLite cannot: positions written by one tenant being invisible to another.
package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestTenantHeaderNamedIsRejected is the recipe of issue #155:
// `curl -H 'X-Tenant-ID: alice' …` answers 400 with code=invalid and a message
// naming the expected format. Before ADR-0005's 2026-09-03 amendment the
// request went through and every named tenant shared tenant 0's rows.
func TestTenantHeaderNamedIsRejected(t *testing.T) {
	ts := newTestServer(t)
	for _, tenant := range []string{"alice", "default", "mon-tenant", "0", "-1", "007", "1.0"} {
		t.Run(tenant, func(t *testing.T) {
			resp := doPost(t, ts.URL+"/v1/metadata.counts", tenant, "{}")
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", resp.StatusCode)
			}
			var env errorEnvelope
			if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
				t.Fatal(err)
			}
			if env.Error.Code != CodeInvalid {
				t.Errorf("code = %q, want %q", env.Error.Code, CodeInvalid)
			}
			if !strings.Contains(env.Error.Message, storage.TenantFormat) {
				t.Errorf("message = %q, want it to name %q", env.Error.Message, storage.TenantFormat)
			}
		})
	}

	// The numeric spelling the proxy is expected to send still passes.
	resp := doPost(t, ts.URL+"/v1/metadata.counts", "42", "{}")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("X-Tenant-ID: 42 → status %d, want 200", resp.StatusCode)
	}
}

// TestTenantIsolationSQLiteScopedFamilies checks two numeric tenants through
// the HTTP layer on the one family the SQLite backend does scope (the filter
// library carries a `scope` column): tenant 1 saves a filter, tenant 2 lists
// none. Positions are not scoped by SQLite — the desktop's single implicit
// tenant — which is why the position-level check lives in the PostgreSQL
// test.
func TestTenantIsolationSQLiteScopedFamilies(t *testing.T) {
	ts := newTestServer(t)

	save := doPost(t, ts.URL+"/v1/filters.save", "1", `{"name":"fav","command":"s"}`)
	defer save.Body.Close()
	if save.StatusCode != http.StatusOK {
		t.Fatalf("tenant 1 filters.save: status %d, want 200", save.StatusCode)
	}

	countFilters := func(tenant string) int {
		t.Helper()
		resp := doPost(t, ts.URL+"/v1/filters.list", tenant, "{}")
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("tenant %s filters.list: status %d, want 200", tenant, resp.StatusCode)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		return len(strings.Fields(string(body))) // one NDJSON object per line, no inner whitespace
	}
	if n := countFilters("1"); n != 1 {
		t.Errorf("tenant 1 sees %d filters, want 1", n)
	}
	if n := countFilters("2"); n != 0 {
		t.Errorf("tenant 2 sees %d of tenant 1's filters, want 0", n)
	}
}
