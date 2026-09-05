// Package metrics is a tiny, dependency-free Prometheus exposition for the
// blunderdb serve daemon. It records an HTTP request counter and a latency
// histogram and renders them in the Prometheus text format.
//
// It deliberately avoids prometheus/client_golang to keep the open-source
// surface lean (no new dependency for the server mode). If richer metrics are
// ever needed, this Registry can be swapped for the official client behind the
// same Middleware/Handler call sites.
package metrics

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// defaultBuckets are the latency histogram upper bounds in seconds. They cover
// the sub-millisecond reads up to multi-second imports.
var defaultBuckets = []float64{
	0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
}

// Registry accumulates HTTP request metrics. It is safe for concurrent use.
type Registry struct {
	buckets []float64

	mu       sync.Mutex
	counters map[counterKey]uint64
	hists    map[histKey]*histogram

	// Rate-limit metrics (atomics: updated outside the request-metrics lock).
	rlRejected uint64 // cumulative requests rejected by the rate limiter
	rlBuckets  uint64 // current number of live per-tenant token buckets

	// PostgreSQL connection-pool gauges (#235), also atomics: polled
	// periodically from a value the pgxpool backend itself owns (Server
	// never holds the pool), so a plain snapshot store fits better than the
	// request-scoped counters/hists above.
	pgHasPool   uint32 // 1 once SetPoolStats has been called at least once (SQLite backend: never)
	pgAcquired  int64
	pgIdle      int64
	pgMax       int64
	pgWaitCount int64

	// Business gauges (#238): in-flight work and database footprint, none of
	// which the request-count/latency metrics above say anything about — a
	// stuck import or a runaway gammonNet sweep is invisible to
	// blunderdb_http_requests_total (it is one long-lived request, not many).
	importsInFlight   int64
	importSpoolBytes  int64
	gammonnetInFlight int64
	hasDBSize         uint32 // 1 once SetDatabaseSizeBytes has been called at least once
	dbSizeBytes       int64
}

// IncRateLimitRejected records one request rejected by the rate limiter.
func (r *Registry) IncRateLimitRejected() {
	if r == nil {
		return
	}
	atomic.AddUint64(&r.rlRejected, 1)
}

// SetRateLimitBuckets records the current number of live per-tenant buckets.
func (r *Registry) SetRateLimitBuckets(n int) {
	if r == nil {
		return
	}
	atomic.StoreUint64(&r.rlBuckets, uint64(n)) //nolint:gosec // G115: n is a live-bucket count (a map length), never negative
}

// SetPoolStats records a snapshot of the PostgreSQL connection pool's state
// (acquired/idle/max connections, and the cumulative count of Acquire calls
// that had to wait for one) — see postgres.Storage.PoolStats, which Server
// polls periodically when the backend implements it. Never called at all
// with the SQLite backend, so WritePrometheus omits the pool gauges rather
// than publish permanent zeros for a pool that does not exist.
func (r *Registry) SetPoolStats(acquired, idle, max int32, waitCount int64) {
	if r == nil {
		return
	}
	atomic.StoreUint32(&r.pgHasPool, 1)
	atomic.StoreInt64(&r.pgAcquired, int64(acquired))
	atomic.StoreInt64(&r.pgIdle, int64(idle))
	atomic.StoreInt64(&r.pgMax, int64(max))
	atomic.StoreInt64(&r.pgWaitCount, waitCount)
}

// SetImportsInFlight records the current number of in-flight
// imports.* jobs, across every tenant (Server.sweepBusinessMetrics polls
// importRegistry's live job count periodically).
func (r *Registry) SetImportsInFlight(n int) {
	if r == nil {
		return
	}
	atomic.StoreInt64(&r.importsInFlight, int64(n))
}

// SetImportSpoolBytes records the current number of bytes reserved from the
// import spool quota (see spoolQuota, handlers_imports.go) — how close the
// daemon is to BLUNDERDB_MAX_SPOOL_BYTES, not just whether an import is
// running at all.
func (r *Registry) SetImportSpoolBytes(n int64) {
	if r == nil {
		return
	}
	atomic.StoreInt64(&r.importSpoolBytes, n)
}

// SetGammonNetSweepsInFlight records the current number of in-flight
// gammonnet.*sweep* jobs, across every tenant.
func (r *Registry) SetGammonNetSweepsInFlight(n int) {
	if r == nil {
		return
	}
	atomic.StoreInt64(&r.gammonnetInFlight, int64(n))
}

// SetDatabaseSizeBytes records the storage backend's current size in bytes
// (the SQLite main file, or PostgreSQL's pg_database_size) — polled
// periodically from whichever backend implements it (see
// server.sizeProvider); a backend that doesn't (there is none today, but the
// duck-typed interface keeps this optional) simply never calls this, and
// WritePrometheus omits the gauge entirely, the same convention the
// PostgreSQL pool gauges already use.
func (r *Registry) SetDatabaseSizeBytes(n int64) {
	if r == nil {
		return
	}
	atomic.StoreUint32(&r.hasDBSize, 1)
	atomic.StoreInt64(&r.dbSizeBytes, n)
}

type counterKey struct {
	method string
	path   string
	status int
}

type histKey struct {
	method string
	path   string
}

type histogram struct {
	counts []uint64 // per-bucket cumulative-eligible counts (raw, summed at render)
	sum    float64
	count  uint64
}

// New returns a Registry with the default latency buckets.
func New() *Registry {
	return &Registry{
		buckets:  defaultBuckets,
		counters: make(map[counterKey]uint64),
		hists:    make(map[histKey]*histogram),
	}
}

// ObserveRequest records one finished HTTP request: its matched route pattern
// (path), method, response status, and duration.
func (r *Registry) ObserveRequest(method, path string, status int, dur time.Duration) {
	if r == nil {
		return
	}
	if path == "" {
		path = "unmatched"
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	r.counters[counterKey{method, path, status}]++

	hk := histKey{method, path}
	h := r.hists[hk]
	if h == nil {
		h = &histogram{counts: make([]uint64, len(r.buckets))}
		r.hists[hk] = h
	}
	secs := dur.Seconds()
	h.sum += secs
	h.count++
	for i, ub := range r.buckets {
		if secs <= ub {
			h.counts[i]++
		}
	}
}

// WritePrometheus renders the accumulated metrics in the Prometheus text
// exposition format.
func (r *Registry) WritePrometheus(w io.Writer) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, _ = fmt.Fprintln(w, "# HELP blunderdb_http_requests_total Total HTTP requests handled.")
	_, _ = fmt.Fprintln(w, "# TYPE blunderdb_http_requests_total counter")
	ckeys := make([]counterKey, 0, len(r.counters))
	for k := range r.counters {
		ckeys = append(ckeys, k)
	}
	sort.Slice(ckeys, func(i, j int) bool {
		a, b := ckeys[i], ckeys[j]
		if a.path != b.path {
			return a.path < b.path
		}
		if a.method != b.method {
			return a.method < b.method
		}
		return a.status < b.status
	})
	for _, k := range ckeys {
		_, _ = fmt.Fprintf(w, "blunderdb_http_requests_total{method=%q,path=%q,status=\"%d\"} %d\n",
			k.method, k.path, k.status, r.counters[k])
	}

	_, _ = fmt.Fprintln(w, "# HELP blunderdb_http_request_duration_seconds HTTP request latency.")
	_, _ = fmt.Fprintln(w, "# TYPE blunderdb_http_request_duration_seconds histogram")
	hkeys := make([]histKey, 0, len(r.hists))
	for k := range r.hists {
		hkeys = append(hkeys, k)
	}
	sort.Slice(hkeys, func(i, j int) bool {
		a, b := hkeys[i], hkeys[j]
		if a.path != b.path {
			return a.path < b.path
		}
		return a.method < b.method
	})
	for _, k := range hkeys {
		h := r.hists[k]
		// counts[i] holds the number of observations <= buckets[i]; because
		// every observation increments each bucket it falls under, counts[i]
		// is already the cumulative ("le") count Prometheus expects.
		for i, ub := range r.buckets {
			le := strconv.FormatFloat(ub, 'g', -1, 64)
			_, _ = fmt.Fprintf(w, "blunderdb_http_request_duration_seconds_bucket{method=%q,path=%q,le=%q} %d\n",
				k.method, k.path, le, h.counts[i])
		}
		_, _ = fmt.Fprintf(w, "blunderdb_http_request_duration_seconds_bucket{method=%q,path=%q,le=\"+Inf\"} %d\n",
			k.method, k.path, h.count)
		_, _ = fmt.Fprintf(w, "blunderdb_http_request_duration_seconds_sum{method=%q,path=%q} %g\n",
			k.method, k.path, h.sum)
		_, _ = fmt.Fprintf(w, "blunderdb_http_request_duration_seconds_count{method=%q,path=%q} %d\n",
			k.method, k.path, h.count)
	}

	_, _ = fmt.Fprintln(w, "# HELP blunderdb_ratelimit_rejected_total Requests rejected by the per-tenant rate limiter.")
	_, _ = fmt.Fprintln(w, "# TYPE blunderdb_ratelimit_rejected_total counter")
	_, _ = fmt.Fprintf(w, "blunderdb_ratelimit_rejected_total %d\n", atomic.LoadUint64(&r.rlRejected))

	_, _ = fmt.Fprintln(w, "# HELP blunderdb_ratelimit_buckets Live per-tenant token buckets.")
	_, _ = fmt.Fprintln(w, "# TYPE blunderdb_ratelimit_buckets gauge")
	_, _ = fmt.Fprintf(w, "blunderdb_ratelimit_buckets %d\n", atomic.LoadUint64(&r.rlBuckets))

	_, _ = fmt.Fprintln(w, "# HELP blunderdb_imports_inflight In-flight imports.* jobs, across every tenant.")
	_, _ = fmt.Fprintln(w, "# TYPE blunderdb_imports_inflight gauge")
	_, _ = fmt.Fprintf(w, "blunderdb_imports_inflight %d\n", atomic.LoadInt64(&r.importsInFlight))

	_, _ = fmt.Fprintln(w, "# HELP blunderdb_import_spool_bytes Bytes currently reserved from the import spool quota.")
	_, _ = fmt.Fprintln(w, "# TYPE blunderdb_import_spool_bytes gauge")
	_, _ = fmt.Fprintf(w, "blunderdb_import_spool_bytes %d\n", atomic.LoadInt64(&r.importSpoolBytes))

	_, _ = fmt.Fprintln(w, "# HELP blunderdb_gammonnet_sweep_inflight In-flight gammonNet catch-up sweeps, across every tenant.")
	_, _ = fmt.Fprintln(w, "# TYPE blunderdb_gammonnet_sweep_inflight gauge")
	_, _ = fmt.Fprintf(w, "blunderdb_gammonnet_sweep_inflight %d\n", atomic.LoadInt64(&r.gammonnetInFlight))

	// Only published once the backend has actually reported a size (see
	// SetDatabaseSizeBytes) — the same convention as the pg_pool_* gauges
	// below, so a backend without a meaningful notion of "database size"
	// never publishes a misleading permanent zero.
	if atomic.LoadUint32(&r.hasDBSize) == 1 {
		_, _ = fmt.Fprintln(w, "# HELP blunderdb_database_size_bytes Storage backend size in bytes (SQLite main file, or PostgreSQL's pg_database_size).")
		_, _ = fmt.Fprintln(w, "# TYPE blunderdb_database_size_bytes gauge")
		_, _ = fmt.Fprintf(w, "blunderdb_database_size_bytes %d\n", atomic.LoadInt64(&r.dbSizeBytes))
	}

	// Only the PostgreSQL backend ever calls SetPoolStats; the SQLite
	// backend has no pool to report, so these gauges are omitted entirely
	// rather than published as permanent, misleading zeros.
	if atomic.LoadUint32(&r.pgHasPool) == 1 {
		_, _ = fmt.Fprintln(w, "# HELP blunderdb_pg_pool_acquired Connections currently checked out of the PostgreSQL pool.")
		_, _ = fmt.Fprintln(w, "# TYPE blunderdb_pg_pool_acquired gauge")
		_, _ = fmt.Fprintf(w, "blunderdb_pg_pool_acquired %d\n", atomic.LoadInt64(&r.pgAcquired))

		_, _ = fmt.Fprintln(w, "# HELP blunderdb_pg_pool_idle Idle connections available in the PostgreSQL pool.")
		_, _ = fmt.Fprintln(w, "# TYPE blunderdb_pg_pool_idle gauge")
		_, _ = fmt.Fprintf(w, "blunderdb_pg_pool_idle %d\n", atomic.LoadInt64(&r.pgIdle))

		_, _ = fmt.Fprintln(w, "# HELP blunderdb_pg_pool_max Configured maximum size of the PostgreSQL pool.")
		_, _ = fmt.Fprintln(w, "# TYPE blunderdb_pg_pool_max gauge")
		_, _ = fmt.Fprintf(w, "blunderdb_pg_pool_max %d\n", atomic.LoadInt64(&r.pgMax))

		_, _ = fmt.Fprintln(w, "# HELP blunderdb_pg_pool_wait_count Cumulative Acquire calls that had to wait for a free connection.")
		_, _ = fmt.Fprintln(w, "# TYPE blunderdb_pg_pool_wait_count counter")
		_, _ = fmt.Fprintf(w, "blunderdb_pg_pool_wait_count %d\n", atomic.LoadInt64(&r.pgWaitCount))
	}
}
