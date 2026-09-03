package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterBurstAndRefill(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	clock := func() time.Time { return now }
	rl := NewRateLimiter(5, 3, clock) // 5 rps, burst 3

	// Burst: the first 3 are allowed, the 4th is rejected (no refill yet).
	for i := 0; i < 3; i++ {
		if !rl.Allow("a") {
			t.Fatalf("request %d should be allowed within burst", i+1)
		}
	}
	if rl.Allow("a") {
		t.Fatal("4th request should be rejected (bucket empty)")
	}

	// After 1s, 5 tokens regenerate but cap at burst (3) → 3 more allowed.
	now = now.Add(time.Second)
	for i := 0; i < 3; i++ {
		if !rl.Allow("a") {
			t.Fatalf("post-refill request %d should be allowed", i+1)
		}
	}
	if rl.Allow("a") {
		t.Fatal("request after refilled burst should be rejected")
	}
}

func TestRateLimiterPerTenantIndependent(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	rl := NewRateLimiter(1, 2, func() time.Time { return now })

	// Exhaust tenant A.
	for i := 0; i < 2; i++ {
		if !rl.Allow("a") {
			t.Fatalf("tenant a request %d should pass", i+1)
		}
	}
	if rl.Allow("a") {
		t.Fatal("tenant a should now be throttled")
	}
	// Tenant B is unaffected.
	for i := 0; i < 2; i++ {
		if !rl.Allow("b") {
			t.Fatalf("tenant b request %d should pass (independent bucket)", i+1)
		}
	}
}

func TestRateLimiterSweep(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	rl := NewRateLimiter(10, 10, func() time.Time { return now })

	for i := 0; i < 1000; i++ {
		rl.Allow(string(rune('A'+i%26)) + string(rune('0'+i/26)))
	}
	if got := rl.Len(); got == 0 {
		t.Fatal("expected live buckets before sweep")
	}

	// Advance past the idle window; every bucket is now stale.
	now = now.Add(time.Hour)
	if got := rl.Sweep(30 * time.Minute); got != 0 {
		t.Fatalf("Sweep left %d buckets, want 0", got)
	}
	if got := rl.Len(); got != 0 {
		t.Fatalf("Len after sweep = %d, want 0", got)
	}
}

func TestRateLimiterBurstFloor(t *testing.T) {
	// burst < 1 is clamped to 1 so a limiter always allows at least one request.
	rl := NewRateLimiter(0.5, 0, func() time.Time { return time.Unix(0, 0) })
	if !rl.Allow("a") {
		t.Fatal("first request should pass with clamped burst of 1")
	}
	if rl.Allow("a") {
		t.Fatal("second request should be throttled")
	}
}

// TestRateLimiterMaxBucketsEvictsLRU guards the hard cap (#230): without it,
// a client sending many distinct X-Tenant-ID values grows the bucket map
// without bound between the periodic Sweep calls. A tiny cap here (via
// newRateLimiterCapped, production always uses DefaultMaxBuckets) makes the
// eviction observable in a handful of requests.
func TestRateLimiterMaxBucketsEvictsLRU(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	rl := newRateLimiterCapped(1, 1, func() time.Time { return now }, 2) // cap of 2 buckets

	// Exhaust tenant "a"'s single-token bucket, and touch it again so it is
	// the most-recently-used of the two entries below.
	if !rl.Allow("a") {
		t.Fatal("first request for a should pass")
	}
	if rl.Allow("a") {
		t.Fatal("second immediate request for a should be throttled (bucket empty)")
	}

	// Fill the second (and last) slot with tenant "b".
	rl.Allow("b")
	if got := rl.Len(); got != 2 {
		t.Fatalf("Len = %d, want 2 (at cap)", got)
	}

	// A third tenant "c" breaches the cap: the least-recently-used bucket
	// ("a", not touched since before "b" arrived) is evicted to make room,
	// and the cap still holds afterwards.
	rl.Allow("c")
	if got := rl.Len(); got != 2 {
		t.Fatalf("Len after cap breach = %d, want 2 (hard cap enforced, not just eventual Sweep)", got)
	}

	// Tenant "a" was evicted: its next request starts a brand new, full
	// bucket and is allowed — even though, had the original bucket survived,
	// it would still read empty from the exhaustion above.
	if !rl.Allow("a") {
		t.Fatal("tenant a should have a fresh bucket after LRU eviction")
	}
}
