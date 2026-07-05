//go:build postgres

// TestTenantPurgeEndpoint provisions a real PostgreSQL via testcontainers-go
// and therefore needs Docker, exactly like every other postgres-tagged test
// in this module (see pkg/blunderdb/storage/postgres/purge_postgres_test.go):
//
//	go test -tags postgres ./internal/server/... -run TestTenantPurge -v
//
// The SQLite "not supported" counterpart, TestTenantPurgeSQLiteNotSupported,
// never touches Postgres and lives in its own untagged file
// (handlers_tenant_sqlite_test.go) so it still runs on the default,
// no-Docker `go test ./...` path.
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

// postTenant issues a POST to the daemon with the given tenant header and an
// optional JSON body (nil for none), mirroring the package-level post helper
// (handlers_domain_test.go) but letting the caller pick the tenant scope
// instead of the fixed testTenant constant.
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
// POST /v1/tenant.purge, and confirms both the HTTP response ({"ok":true})
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

	purgeResp := postTenant(t, ts, "1", "/v1/tenant.purge", nil)
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

	loadResp := postTenant(t, ts, "1", "/v1/positions.load", idReq{ID: saved.ID})
	defer loadResp.Body.Close()
	if loadResp.StatusCode != http.StatusNotFound {
		t.Fatalf("post-purge load status = %d, want 404 (position should be gone)", loadResp.StatusCode)
	}
}
