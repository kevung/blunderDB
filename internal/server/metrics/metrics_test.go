package metrics

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestObserveRequest_CountsByMethodPathStatus(t *testing.T) {
	r := New()
	r.ObserveRequest("GET", "/healthz", 200, time.Millisecond)
	r.ObserveRequest("GET", "/healthz", 200, time.Millisecond)
	r.ObserveRequest("GET", "/healthz", 503, time.Millisecond)
	r.ObserveRequest("POST", "/v1/positions.list", 200, time.Millisecond)

	want := map[counterKey]uint64{
		{"GET", "/healthz", 200}:            2,
		{"GET", "/healthz", 503}:            1,
		{"POST", "/v1/positions.list", 200}: 1,
	}
	if len(r.counters) != len(want) {
		t.Fatalf("counters = %v, want %v", r.counters, want)
	}
	for k, n := range want {
		if r.counters[k] != n {
			t.Errorf("counters[%+v] = %d, want %d", k, r.counters[k], n)
		}
	}
	// The histogram is keyed without the status: one series per route.
	if len(r.hists) != 2 {
		t.Fatalf("hists has %d keys, want 2 (status is not a histogram label)", len(r.hists))
	}
	if h := r.hists[histKey{"GET", "/healthz"}]; h == nil || h.count != 3 {
		t.Fatalf("hist[GET /healthz] = %+v, want count 3", h)
	}
}

// TestObserveRequest_HistogramIsCumulative pins the bucket semantics
// WritePrometheus relies on: counts[i] is the number of observations whose
// latency is <= buckets[i], so it can be rendered as the "le" series as is.
func TestObserveRequest_HistogramIsCumulative(t *testing.T) {
	r := New()
	r.ObserveRequest("GET", "/x", 200, 500*time.Microsecond) // <= 0.001
	r.ObserveRequest("GET", "/x", 200, 30*time.Millisecond)  // <= 0.05
	r.ObserveRequest("GET", "/x", 200, 20*time.Second)       // above every bucket

	h := r.hists[histKey{"GET", "/x"}]
	if h == nil {
		t.Fatal("no histogram recorded")
	}
	if h.count != 3 {
		t.Errorf("count = %d, want 3", h.count)
	}
	if wantSum := 0.0005 + 0.03 + 20; h.sum != wantSum {
		t.Errorf("sum = %g, want %g", h.sum, wantSum)
	}
	for i, ub := range r.buckets {
		var want uint64
		switch {
		case ub >= 0.05:
			want = 2
		case ub >= 0.001:
			want = 1
		}
		if h.counts[i] != want {
			t.Errorf("bucket le=%g: count = %d, want %d", ub, h.counts[i], want)
		}
	}
}

func TestObserveRequest_EmptyPathIsUnmatched(t *testing.T) {
	r := New()
	r.ObserveRequest("GET", "", 404, time.Millisecond)
	if _, ok := r.counters[counterKey{"GET", "unmatched", 404}]; !ok {
		t.Fatalf("empty path not folded into %q: %v", "unmatched", r.counters)
	}
}

// TestNilRegistry_IsNoOp: the middleware and the rate limiter call these on
// whatever Options.Metrics holds, nil included when metrics are disabled.
func TestNilRegistry_IsNoOp(t *testing.T) {
	var r *Registry
	r.ObserveRequest("GET", "/healthz", 200, time.Millisecond)
	r.IncRateLimitRejected()
	r.SetRateLimitBuckets(3)
}

func TestRateLimitMetrics(t *testing.T) {
	r := New()
	r.IncRateLimitRejected()
	r.IncRateLimitRejected()
	r.SetRateLimitBuckets(7)
	r.SetRateLimitBuckets(4) // a gauge: the last value wins

	var out strings.Builder
	r.WritePrometheus(&out)
	text := out.String()
	for _, line := range []string{
		"blunderdb_ratelimit_rejected_total 2\n",
		"blunderdb_ratelimit_buckets 4\n",
	} {
		if !strings.Contains(text, line) {
			t.Errorf("missing %q in:\n%s", line, text)
		}
	}
}

// TestWritePrometheus_Format is the exposition golden: HELP/TYPE headers,
// quoted labels, one "le" line per bucket plus +Inf, then _sum and _count,
// the two rate-limit series, the business gauges (#238), and — because
// SetDatabaseSizeBytes was called — the database size gauge, in this exact
// order.
func TestWritePrometheus_Format(t *testing.T) {
	r := New()
	r.ObserveRequest("GET", "/healthz", 200, 3*time.Millisecond)
	r.ObserveRequest("GET", "/healthz", 200, 7*time.Millisecond)
	r.SetRateLimitBuckets(1)
	r.SetImportsInFlight(2)
	r.SetImportSpoolBytes(1024)
	r.SetGammonNetSweepsInFlight(1)
	r.SetDatabaseSizeBytes(4096)

	var out strings.Builder
	r.WritePrometheus(&out)

	want := `# HELP blunderdb_http_requests_total Total HTTP requests handled.
# TYPE blunderdb_http_requests_total counter
blunderdb_http_requests_total{method="GET",path="/healthz",status="200"} 2
# HELP blunderdb_http_request_duration_seconds HTTP request latency.
# TYPE blunderdb_http_request_duration_seconds histogram
blunderdb_http_request_duration_seconds_bucket{method="GET",path="/healthz",le="0.001"} 0
blunderdb_http_request_duration_seconds_bucket{method="GET",path="/healthz",le="0.005"} 1
blunderdb_http_request_duration_seconds_bucket{method="GET",path="/healthz",le="0.01"} 2
blunderdb_http_request_duration_seconds_bucket{method="GET",path="/healthz",le="0.025"} 2
blunderdb_http_request_duration_seconds_bucket{method="GET",path="/healthz",le="0.05"} 2
blunderdb_http_request_duration_seconds_bucket{method="GET",path="/healthz",le="0.1"} 2
blunderdb_http_request_duration_seconds_bucket{method="GET",path="/healthz",le="0.25"} 2
blunderdb_http_request_duration_seconds_bucket{method="GET",path="/healthz",le="0.5"} 2
blunderdb_http_request_duration_seconds_bucket{method="GET",path="/healthz",le="1"} 2
blunderdb_http_request_duration_seconds_bucket{method="GET",path="/healthz",le="2.5"} 2
blunderdb_http_request_duration_seconds_bucket{method="GET",path="/healthz",le="5"} 2
blunderdb_http_request_duration_seconds_bucket{method="GET",path="/healthz",le="10"} 2
blunderdb_http_request_duration_seconds_bucket{method="GET",path="/healthz",le="+Inf"} 2
blunderdb_http_request_duration_seconds_sum{method="GET",path="/healthz"} 0.01
blunderdb_http_request_duration_seconds_count{method="GET",path="/healthz"} 2
# HELP blunderdb_ratelimit_rejected_total Requests rejected by the per-tenant rate limiter.
# TYPE blunderdb_ratelimit_rejected_total counter
blunderdb_ratelimit_rejected_total 0
# HELP blunderdb_ratelimit_buckets Live per-tenant token buckets.
# TYPE blunderdb_ratelimit_buckets gauge
blunderdb_ratelimit_buckets 1
# HELP blunderdb_imports_inflight In-flight imports.* jobs, across every tenant.
# TYPE blunderdb_imports_inflight gauge
blunderdb_imports_inflight 2
# HELP blunderdb_import_spool_bytes Bytes currently reserved from the import spool quota.
# TYPE blunderdb_import_spool_bytes gauge
blunderdb_import_spool_bytes 1024
# HELP blunderdb_gammonnet_sweep_inflight In-flight gammonNet catch-up sweeps, across every tenant.
# TYPE blunderdb_gammonnet_sweep_inflight gauge
blunderdb_gammonnet_sweep_inflight 1
# HELP blunderdb_database_size_bytes Storage backend size in bytes (SQLite main file, or PostgreSQL's pg_database_size).
# TYPE blunderdb_database_size_bytes gauge
blunderdb_database_size_bytes 4096
`
	if got := out.String(); got != want {
		t.Errorf("exposition differs.\n--- got ---\n%s--- want ---\n%s", got, want)
	}
}

// TestWritePrometheus_DatabaseSizeOmittedUntilSet mirrors the PostgreSQL pool
// gauges' convention: blunderdb_database_size_bytes must not appear at all
// until SetDatabaseSizeBytes has been called at least once (a backend with
// no meaningful notion of "size" — none exists today, but the interface is
// optional — must never publish a misleading permanent zero).
func TestWritePrometheus_DatabaseSizeOmittedUntilSet(t *testing.T) {
	r := New()
	var out strings.Builder
	r.WritePrometheus(&out)
	if strings.Contains(out.String(), "blunderdb_database_size_bytes") {
		t.Errorf("blunderdb_database_size_bytes must be omitted before SetDatabaseSizeBytes is ever called:\n%s", out.String())
	}
}

// TestWritePrometheus_EmptyRegistryStillExposesHeaders: a freshly started
// daemon must answer /metrics with every family declared, so a scraper sees
// the metric names before the first request.
func TestWritePrometheus_EmptyRegistryStillExposesHeaders(t *testing.T) {
	var out strings.Builder
	New().WritePrometheus(&out)
	text := out.String()
	for _, family := range []string{
		"# TYPE blunderdb_http_requests_total counter",
		"# TYPE blunderdb_http_request_duration_seconds histogram",
		"# TYPE blunderdb_ratelimit_rejected_total counter",
		"# TYPE blunderdb_ratelimit_buckets gauge",
		"# TYPE blunderdb_imports_inflight gauge",
		"# TYPE blunderdb_import_spool_bytes gauge",
		"# TYPE blunderdb_gammonnet_sweep_inflight gauge",
	} {
		if !strings.Contains(text, family) {
			t.Errorf("missing %q in:\n%s", family, text)
		}
	}
	if strings.Contains(text, "blunderdb_http_requests_total{") {
		t.Errorf("empty registry rendered a request series:\n%s", text)
	}
}

// TestWritePrometheus_IsSortedAndStable: series come out ordered by path,
// method, status (maps would otherwise shuffle them), and two renders of the
// same registry are byte-identical.
func TestWritePrometheus_IsSortedAndStable(t *testing.T) {
	r := New()
	r.ObserveRequest("POST", "/v1/positions.list", 200, time.Millisecond)
	r.ObserveRequest("GET", "/readyz", 503, time.Millisecond)
	r.ObserveRequest("GET", "/readyz", 200, time.Millisecond)
	r.ObserveRequest("GET", "/healthz", 200, time.Millisecond)
	r.ObserveRequest("POST", "/healthz", 405, time.Millisecond)

	var first strings.Builder
	r.WritePrometheus(&first)
	var second strings.Builder
	r.WritePrometheus(&second)
	if first.String() != second.String() {
		t.Fatalf("two renders differ:\n%s\n---\n%s", first.String(), second.String())
	}

	var counters []string
	for _, line := range strings.Split(first.String(), "\n") {
		if strings.HasPrefix(line, "blunderdb_http_requests_total{") {
			counters = append(counters, line)
		}
	}
	want := []string{
		`blunderdb_http_requests_total{method="GET",path="/healthz",status="200"} 1`,
		`blunderdb_http_requests_total{method="POST",path="/healthz",status="405"} 1`,
		`blunderdb_http_requests_total{method="GET",path="/readyz",status="200"} 1`,
		`blunderdb_http_requests_total{method="GET",path="/readyz",status="503"} 1`,
		`blunderdb_http_requests_total{method="POST",path="/v1/positions.list",status="200"} 1`,
	}
	if strings.Join(counters, "\n") != strings.Join(want, "\n") {
		t.Errorf("counter order:\n got %v\nwant %v", counters, want)
	}
}

// TestObserveRequest_Concurrent is meant for -race: the registry is shared
// by every request goroutine and by the /metrics scrape.
func TestObserveRequest_Concurrent(t *testing.T) {
	r := New()
	const goroutines, perGoroutine = 8, 500
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				r.ObserveRequest("GET", "/healthz", 200, time.Duration(i)*time.Microsecond)
				if i%100 == 0 {
					var sink strings.Builder
					r.WritePrometheus(&sink)
				}
			}
		}()
	}
	wg.Wait()
	if n := r.counters[counterKey{"GET", "/healthz", 200}]; n != goroutines*perGoroutine {
		t.Fatalf("counter = %d, want %d", n, goroutines*perGoroutine)
	}
	if h := r.hists[histKey{"GET", "/healthz"}]; h.count != goroutines*perGoroutine {
		t.Fatalf("histogram count = %d, want %d", h.count, goroutines*perGoroutine)
	}
}
