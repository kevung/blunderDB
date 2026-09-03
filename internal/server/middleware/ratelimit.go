package middleware

import (
	"container/list"
	"net/http"
	"sync"
	"time"
)

// DefaultMaxBuckets hard-caps the number of live per-tenant buckets a
// RateLimiter built by NewRateLimiter will hold. Without a cap, a client that
// sends many distinct X-Tenant-ID values — accidentally or as an attack —
// grows the bucket map without bound between the periodic Sweep calls
// (server.go runs one every 5 minutes); Allow instead evicts the
// least-recently-used bucket the instant a new tenant would exceed the cap,
// so the map's memory footprint is bounded on every single request, not just
// eventually.
const DefaultMaxBuckets = 10_000

// RateLimiter is a per-tenant token-bucket limiter. Each tenant gets an
// independent bucket that refills at rps tokens/second up to burst tokens.
// It is dependency-free (no golang.org/x/time/rate) and safe for concurrent
// use. The clock is injectable for deterministic tests.
type RateLimiter struct {
	rps        float64
	burst      float64
	now        func() time.Time
	maxBuckets int

	mu      sync.Mutex
	buckets map[string]*list.Element // tenant -> element wrapping *tokenBucket
	order   *list.List               // front = most recently used
}

type tokenBucket struct {
	tenant   string
	tokens   float64
	last     time.Time
	lastSeen time.Time
}

// NewRateLimiter builds a limiter allowing rps requests/second per tenant with
// the given burst, capped at DefaultMaxBuckets live tenants. now defaults to
// time.Now when nil.
func NewRateLimiter(rps float64, burst int, now func() time.Time) *RateLimiter {
	return newRateLimiterCapped(rps, burst, now, DefaultMaxBuckets)
}

// newRateLimiterCapped is NewRateLimiter with an explicit bucket cap. It
// exists so tests can exercise LRU eviction without creating DefaultMaxBuckets
// buckets; production code always goes through NewRateLimiter.
func newRateLimiterCapped(rps float64, burst int, now func() time.Time, maxBuckets int) *RateLimiter {
	if now == nil {
		now = time.Now
	}
	if burst < 1 {
		burst = 1
	}
	if maxBuckets < 1 {
		maxBuckets = 1
	}
	return &RateLimiter{
		rps:        rps,
		burst:      float64(burst),
		now:        now,
		maxBuckets: maxBuckets,
		buckets:    make(map[string]*list.Element),
		order:      list.New(),
	}
}

// Allow reports whether a request for tenant may proceed, consuming one token
// when it does.
func (rl *RateLimiter) Allow(tenant string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()

	var b *tokenBucket
	if el, ok := rl.buckets[tenant]; ok {
		rl.order.MoveToFront(el)
		b = el.Value.(*tokenBucket)
		// Refill since the last observation, capped at burst.
		if elapsed := now.Sub(b.last).Seconds(); elapsed > 0 {
			b.tokens += elapsed * rl.rps
			if b.tokens > rl.burst {
				b.tokens = rl.burst
			}
			b.last = now
		}
	} else {
		if len(rl.buckets) >= rl.maxBuckets {
			rl.evictOldestLocked()
		}
		b = &tokenBucket{tenant: tenant, tokens: rl.burst, last: now}
		rl.buckets[tenant] = rl.order.PushFront(b)
	}
	b.lastSeen = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// evictOldestLocked drops the least-recently-used bucket. Callers must hold
// rl.mu.
func (rl *RateLimiter) evictOldestLocked() {
	oldest := rl.order.Back()
	if oldest == nil {
		return
	}
	b := oldest.Value.(*tokenBucket)
	delete(rl.buckets, b.tenant)
	rl.order.Remove(oldest)
}

// Len returns the number of live per-tenant buckets.
func (rl *RateLimiter) Len() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return len(rl.buckets)
}

// Sweep evicts buckets not seen within maxIdle, so idle tenants don't bloat the
// map. It returns the number of buckets remaining.
func (rl *RateLimiter) Sweep(maxIdle time.Duration) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := rl.now().Add(-maxIdle)
	for tenant, el := range rl.buckets {
		b := el.Value.(*tokenBucket)
		if b.lastSeen.Before(cutoff) {
			delete(rl.buckets, tenant)
			rl.order.Remove(el)
		}
	}
	return len(rl.buckets)
}

// RateLimit rejects requests once a tenant exceeds its bucket. The tenant is
// read from the context (set by Tenant, which must run first); requests without
// a tenant are passed through untouched. onReject writes the response for a
// throttled request (the server supplies the rate_limited error envelope);
// Retry-After is set before it is called.
func RateLimit(rl *RateLimiter, onReject func(http.ResponseWriter, *http.Request)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenant, ok := TenantFromContext(r.Context())
			if !ok || tenant == "" {
				next.ServeHTTP(w, r)
				return
			}
			if !rl.Allow(tenant) {
				w.Header().Set("Retry-After", "1")
				onReject(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
