package main

import (
	"context"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kevung/blunderdb/internal/server"
	"github.com/kevung/blunderdb/internal/server/metrics"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// newDaemon starts the real daemon (the same handler chain `blunderdb serve`
// runs, on an in-memory SQLite) so the load generator is exercised against
// the tenant middleware it is meant to probe, not a stub.
func newDaemon(t *testing.T) *httptest.Server {
	t.Helper()
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	srv, err := server.New(server.Options{Storage: st, Metrics: metrics.New()})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

// TestScenarioTenantExpectations: only named-tenants expects the 400 refusal
// (ADR-0005).
func TestScenarioTenantExpectations(t *testing.T) {
	ts := newDaemon(t)
	rng := rand.New(rand.NewSource(1))
	client := ts.Client()

	numeric := scenarios["mixed"]
	if s := doRequest(client, ts.URL, numeric.tenant(7), numeric.ok, op{"positions.list", 1, buildList}, rng); s.fail {
		t.Errorf("mixed scenario, tenant %q: request counted as a failure", numeric.tenant(7))
	}

	named := scenarios["named-tenants"]
	if got := named.tenant(7); got != "tenant-7" {
		t.Fatalf("named tenant = %q, want %q", got, "tenant-7")
	}
	if s := doRequest(client, ts.URL, named.tenant(7), named.ok, op{"positions.list", 1, buildList}, rng); s.fail {
		t.Errorf("named-tenants scenario: the daemon did not refuse %q with 400", named.tenant(7))
	}
	// The same refusal is an error in every numeric scenario.
	if s := doRequest(client, ts.URL, named.tenant(7), numeric.ok, op{"positions.list", 1, buildList}, rng); !s.fail {
		t.Errorf("mixed scenario: a 400 for %q was not counted as a failure", named.tenant(7))
	}

	// The raw contract, independent of the sample bookkeeping.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/positions.list", jsonBody([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "alice")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("X-Tenant-ID: alice → %d, want 400", resp.StatusCode)
	}
}

// TestRunNamedTenantsScenario drives a short closed-loop run of the
// named-tenants scenario end to end and expects a clean report: every
// request was refused, and every refusal was the expected outcome.
func TestRunNamedTenantsScenario(t *testing.T) {
	ts := newDaemon(t)
	cfg := config{
		target:      ts.URL,
		tenants:     5,
		duration:    150 * time.Millisecond,
		scenario:    "named-tenants",
		concurrency: 2,
		seed:        1,
	}
	rep, err := run(cfg, scenarios[cfg.scenario])
	if err != nil {
		t.Fatal(err)
	}
	if rep.TotalRequests == 0 {
		t.Fatal("no request issued")
	}
	if rep.Errors != 0 {
		t.Errorf("named-tenants: %d of %d requests were not refused with 400", rep.Errors, rep.TotalRequests)
	}
}

// TestDoRequestRetriesBackpressure: a 503 is retried, honouring Retry-After,
// and reaches the report as back-pressure, never as an error; once the retries
// run out it is shed, still not an error.
func TestDoRequestRetriesBackpressure(t *testing.T) {
	oldMax, oldBase, oldCap := maxRetries, baseWait, maxWait
	maxRetries, baseWait, maxWait = 3, time.Millisecond, 5*time.Millisecond
	t.Cleanup(func() { maxRetries, baseWait, maxWait = oldMax, oldBase, oldCap })

	calls := 0
	busy := 2 // answers 503 this many times, then 200
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls <= busy {
			w.Header().Set("Retry-After", "1") // capped by maxWait
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	o := op{name: "x", build: func(*rand.Rand) (string, []byte) { return "/x", []byte(`{}`) }}
	rng := rand.New(rand.NewSource(1))
	s := doRequest(ts.Client(), ts.URL, "1", is2xx, o, rng)
	if s.fail || s.shed || s.retries != 2 || calls != 3 {
		t.Fatalf("recovered request: %+v after %d calls", s, calls)
	}

	calls, busy = 0, 1000
	s = doRequest(ts.Client(), ts.URL, "1", is2xx, o, rng)
	if s.fail || !s.shed || s.retries != 3 || calls != 4 {
		t.Fatalf("exhausted request: %+v after %d calls", s, calls)
	}

	rep := buildReport(config{}, time.Second, []sample{s, {op: "x", fail: true, code: "500 ?"}})
	if rep.Errors != 1 || rep.Backpressure != (backpressure{Requests: 1, Retries: 3, Shed: 1}) {
		t.Fatalf("report: errors=%d bp=%+v", rep.Errors, rep.Backpressure)
	}
}

func TestRetryWait(t *testing.T) {
	if got := retryWait(0, "1"); got != time.Second {
		t.Fatalf("Retry-After honoured: %v", got)
	}
	if got := retryWait(1, ""); got != 2*baseWait {
		t.Fatalf("exponential step: %v", got)
	}
	if got := retryWait(0, "3600"); got != maxWait {
		t.Fatalf("capped: %v", got)
	}
}
