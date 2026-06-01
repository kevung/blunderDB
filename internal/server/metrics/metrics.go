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

	fmt.Fprintln(w, "# HELP blunderdb_http_requests_total Total HTTP requests handled.")
	fmt.Fprintln(w, "# TYPE blunderdb_http_requests_total counter")
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
		fmt.Fprintf(w, "blunderdb_http_requests_total{method=%q,path=%q,status=\"%d\"} %d\n",
			k.method, k.path, k.status, r.counters[k])
	}

	fmt.Fprintln(w, "# HELP blunderdb_http_request_duration_seconds HTTP request latency.")
	fmt.Fprintln(w, "# TYPE blunderdb_http_request_duration_seconds histogram")
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
			fmt.Fprintf(w, "blunderdb_http_request_duration_seconds_bucket{method=%q,path=%q,le=%q} %d\n",
				k.method, k.path, le, h.counts[i])
		}
		fmt.Fprintf(w, "blunderdb_http_request_duration_seconds_bucket{method=%q,path=%q,le=\"+Inf\"} %d\n",
			k.method, k.path, h.count)
		fmt.Fprintf(w, "blunderdb_http_request_duration_seconds_sum{method=%q,path=%q} %g\n",
			k.method, k.path, h.sum)
		fmt.Fprintf(w, "blunderdb_http_request_duration_seconds_count{method=%q,path=%q} %d\n",
			k.method, k.path, h.count)
	}
}
