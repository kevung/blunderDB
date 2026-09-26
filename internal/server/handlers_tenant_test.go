//go:build postgres

// TestTenantPurgeEndpoint needs Docker (testcontainers-go), like every
// postgres-tagged test:
//
//	go test -tags postgres ./internal/server/... -run TestTenantPurge -v
//
// The SQLite counterpart lives untagged in handlers_tenant_sqlite_test.go so
// it runs on the default `go test ./...`.
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/internal/server/metrics"
	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
	"github.com/testcontainers/testcontainers-go"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// newPostgresTestServer builds a Server backed by a fresh PostgreSQL 16
// testcontainer. The test is skipped (not failed) when Docker is unavailable,
// matching startPostgres/purgeTestDB's convention in the postgres package.
func newPostgresTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	ctx := context.Background()

	container, err := tcpg.Run(ctx, "postgres:16-alpine",
		tcpg.WithDatabase("blunderdb"),
		tcpg.WithUsername("test"),
		tcpg.WithPassword("test"),
		tcpg.BasicWaitStrategies(),
	)
	if err != nil {
		t.Skipf("postgres container unavailable (Docker required): %v", err)
	}
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(container) })

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	st, err := pg.Open(ctx, dsn, nil)
	if err != nil {
		t.Fatalf("pg.Open: %v", err)
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

// postTenant POSTs body (nil for none) under the given tenant header.
func postTenant(t *testing.T, ts *httptest.Server, tenant, path string, body any) *http.Response {
	t.Helper()
	var reader *strings.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = strings.NewReader(string(buf))
	} else {
		reader = strings.NewReader("")
	}
	req, _ := http.NewRequest(http.MethodPost, ts.URL+path, reader)
	req.Header.Set(middleware.TenantHeader, tenant)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// TestTenantPurgeEndpoint seeds a position for tenant "1", purges it through
// POST /ops/tenant.purge, and confirms both the HTTP response ({"ok":true})
// and that the position is actually gone (a subsequent positions.load 404s).
func TestTenantPurgeEndpoint(t *testing.T) {
	ts := newPostgresTestServer(t)

	p := domain.InitializePosition()
	saveResp := postTenant(t, ts, "1", "/v1/positions.save", positionReq{Position: &p})
	defer saveResp.Body.Close()
	if saveResp.StatusCode != http.StatusOK {
		t.Fatalf("seed save status = %d, want 200", saveResp.StatusCode)
	}
	var saved idResp
	if err := json.NewDecoder(saveResp.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if saved.ID == 0 {
		t.Fatal("seed save returned id 0")
	}

	purgeResp := postTenant(t, ts, "1", "/ops/tenant.purge", nil)
	defer purgeResp.Body.Close()
	if purgeResp.StatusCode != http.StatusOK {
		t.Fatalf("purge status = %d, want 200", purgeResp.StatusCode)
	}
	var purgeBody okResp
	if err := json.NewDecoder(purgeResp.Body).Decode(&purgeBody); err != nil {
		t.Fatal(err)
	}
	if !purgeBody.OK {
		t.Fatalf("purge body.OK = %v, want true", purgeBody.OK)
	}

	loadResp := postTenant(t, ts, "1", "/v1/positions.load", idReq(saved))
	defer loadResp.Body.Close()
	if loadResp.StatusCode != http.StatusNotFound {
		t.Fatalf("post-purge load status = %d, want 404 (position should be gone)", loadResp.StatusCode)
	}
}

// TestTenantIsolationNamed: tenant 2 gets 404 on tenant 1's position, and a
// named tenant ("alice") is refused with 400 code=invalid (ADR-0005).
func TestTenantIsolationNamed(t *testing.T) {
	ts := newPostgresTestServer(t)

	p := domain.InitializePosition()
	saveResp := postTenant(t, ts, "1", "/v1/positions.save", positionReq{Position: &p})
	defer saveResp.Body.Close()
	if saveResp.StatusCode != http.StatusOK {
		t.Fatalf("tenant 1 save status = %d, want 200", saveResp.StatusCode)
	}
	var saved idResp
	if err := json.NewDecoder(saveResp.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}

	ownResp := postTenant(t, ts, "1", "/v1/positions.load", idReq(saved))
	defer ownResp.Body.Close()
	if ownResp.StatusCode != http.StatusOK {
		t.Fatalf("tenant 1 load of its own position: status %d, want 200", ownResp.StatusCode)
	}

	otherResp := postTenant(t, ts, "2", "/v1/positions.load", idReq(saved))
	defer otherResp.Body.Close()
	if otherResp.StatusCode != http.StatusNotFound {
		t.Fatalf("tenant 2 load of tenant 1's position: status %d, want 404", otherResp.StatusCode)
	}

	for _, named := range []string{"alice", "bob", "default"} {
		resp := postTenant(t, ts, named, "/v1/positions.load", idReq(saved))
		var env errorEnvelope
		err := json.NewDecoder(resp.Body).Decode(&env)
		resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusBadRequest || env.Error.Code != CodeInvalid {
			t.Errorf("X-Tenant-ID %q: status %d code %q, want 400 %q", named, resp.StatusCode, env.Error.Code, CodeInvalid)
		}
	}
}

// TestSessionIsolationPostgres: session_state is keyed by tenant_id, so one
// tenant neither loads nor clears another's session.
func TestSessionIsolationPostgres(t *testing.T) {
	ts := newPostgresTestServer(t)
	assertSessionIsolated(t, ts)
}
