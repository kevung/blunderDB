package middleware

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter is a per-tenant token-bucket limiter. Each tenant gets an
// independent bucket that refills at rps tokens/second up to burst tokens.
// It is dependency-free (no golang.org/x/time/rate) and safe for concurrent
// use. The clock is injectable for deterministic tests.
type RateLimiter struct {
	rps   float64
	burst float64
	now   func() time.Time

	mu      sync.Mutex
	buckets map[string]*tokenBucket
}

type tokenBucket struct {
	tokens   float64
	last     time.Time
	lastSeen time.Time
}

// NewRateLimiter builds a limiter allowing rps requests/second per tenant with
// the given burst. now defaults to time.Now when nil.
func NewRateLimiter(rps float64, burst int, now func() time.Time) *RateLimiter {
	if now == nil {
		now = time.Now
	}
	if burst < 1 {
		burst = 1
	}
	return &RateLimiter{
		rps:     rps,
		burst:   float64(burst),
		now:     now,
		buckets: make(map[string]*tokenBucket),
	}
}

// Allow reports whether a request for tenant may proceed, consuming one token
// when it does.
func (rl *RateLimiter) Allow(tenant string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()
	b := rl.buckets[tenant]
	if b == nil {
		b = &tokenBucket{tokens: rl.burst, last: now}
		rl.buckets[tenant] = b
	}

	// Refill since the last observation, capped at burst.
	if elapsed := now.Sub(b.last).Seconds(); elapsed > 0 {
		b.tokens += elapsed * rl.rps
		if b.tokens > rl.burst {
			b.tokens = rl.burst
		}
		b.last = now
	}
	b.lastSeen = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
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
	for tenant, b := range rl.buckets {
		if b.lastSeen.Before(cutoff) {
			delete(rl.buckets, tenant)
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
