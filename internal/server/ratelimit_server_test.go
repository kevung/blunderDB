package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/internal/server/metrics"
	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// rateLimitedServer builds an httptest server with rate limiting on and a frozen
// clock (so buckets never refill during the test).
func rateLimitedServer(t *testing.T, rps float64, burst int) (*httptest.Server, *metrics.Registry) {
	t.Helper()
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	reg := metrics.New()
	frozen := time.Unix(1_700_000_000, 0)
	srv, err := New(Options{
		Storage:        st,
		Metrics:        reg,
		EnableMetrics:  true,
		RateLimitRPS:   rps,
		RateLimitBurst: burst,
		now:            func() time.Time { return frozen },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, reg
}

func callCounts(t *testing.T, ts *httptest.Server, tenant string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/metadata.counts", strings.NewReader("{}"))
	req.Header.Set(middleware.TenantHeader, tenant)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestServerRateLimitThrottles(t *testing.T) {
	ts, reg := rateLimitedServer(t, 1, 2) // burst 2, frozen clock → no refill

	// First two requests for tenant "a" pass.
	for i := 0; i < 2; i++ {
		resp := callCounts(t, ts, "a")
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("request %d: status %d, want 200", i+1, resp.StatusCode)
		}
	}

	// The third is throttled with the rate_limited envelope and Retry-After.
	resp := callCounts(t, ts, "a")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("3rd request: status %d, want 429", resp.StatusCode)
	}
	if resp.Header.Get("Retry-After") == "" {
		t.Error("missing Retry-After header on 429")
	}
	var env struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if env.Error.Code != CodeRateLimited {
		t.Fatalf("error code = %q, want %q", env.Error.Code, CodeRateLimited)
	}

	// A different tenant has its own bucket and is unaffected.
	respB := callCounts(t, ts, "b")
	respB.Body.Close()
	if respB.StatusCode != http.StatusOK {
		t.Fatalf("tenant b: status %d, want 200 (independent bucket)", respB.StatusCode)
	}

	// The rejection is reflected in /metrics.
	var sb strings.Builder
	reg.WritePrometheus(&sb)
	if !strings.Contains(sb.String(), "blunderdb_ratelimit_rejected_total 1") {
		t.Errorf("metrics missing the rejection counter:\n%s", sb.String())
	}
}

func TestServerRateLimitDisabledByDefault(t *testing.T) {
	ts := newTestServer(t) // no RateLimitRPS → middleware not mounted
	for i := 0; i < 50; i++ {
		resp := callCounts(t, ts, "a")
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("request %d throttled unexpectedly: status %d", i+1, resp.StatusCode)
		}
	}
}
