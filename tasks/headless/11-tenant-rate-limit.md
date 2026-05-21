# Phase C (optional) — Per-tenant rate-limiting middleware

**Goal.** Protect the `serve` daemon against a single tenant exhausting
shared resources. Disabled by default; opt-in via flag.

**Estimate.** 1-2 days. **Risk.** Low. **PRs.** 1.

**Prerequisites.** [P6](06-serve-http.md).

## Why

In a 10 k-tenant deployment, one misbehaving tenant (a bug, a runaway
script) can consume all DB connections, fill the request queue, and
degrade everyone's latency. A simple token-bucket per tenant caps the
damage.

This is **optional** because:
- The upstream reverse-proxy may already enforce per-client rate limits.
- It introduces operational complexity (tuning, observability) best
  deferred until a real incident motivates it.

## Design

`internal/server/middleware/ratelimit.go`:

```go
type RateLimiter struct {
    rps   float64       // tokens per second per tenant
    burst int
    mu    sync.Mutex
    buckets map[string]*rate.Limiter   // golang.org/x/time/rate
}

func (rl *RateLimiter) Allow(tenant string) bool {
    rl.mu.Lock()
    l, ok := rl.buckets[tenant]
    if !ok {
        l = rate.NewLimiter(rate.Limit(rl.rps), rl.burst)
        rl.buckets[tenant] = l
    }
    rl.mu.Unlock()
    return l.Allow()
}
```

Middleware:

```go
func RateLimit(rl *RateLimiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            tenant := tenantFromCtx(r.Context())
            if !rl.Allow(tenant) {
                w.Header().Set("Retry-After", "1")
                writeError(w, errors.New("rate_limit_exceeded"), 429)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

## Configuration

`blunderdb serve` flags:

```
--rate-limit-rps    <float>   [default 0 = disabled]
--rate-limit-burst  <int>     [default 2 × rps]
```

If `--rate-limit-rps 0`, the middleware is **not** mounted (no overhead).
For 10 k tenants × 1 req/s sustained, a per-tenant cap of ~5 RPS with
burst 20 is a reasonable starting point.

## Cleanup

Idle tenants must not bloat the map indefinitely. A background goroutine
sweeps every 5 min and evicts buckets unused for > 30 min.

Add `rl_buckets` Prometheus gauge for observability.

## Error code

Extend the closed code set introduced in P6:

```
{"error":{"code":"rate_limited","message":"too many requests"}}
```

HTTP status: 429.

This **does** add a new code to the API. Document in `pkg/blunderdb/api/doc.go`
and bump the API minor version if you tag releases.

## Steps

- [ ] Add `golang.org/x/time/rate` to `go.mod` (small, stdlib-adjacent).
- [ ] Implement `RateLimiter` and `RateLimit` middleware.
- [ ] Wire into the server pipeline conditionally on `--rate-limit-rps`.
- [ ] Add `rl_buckets` gauge + `rl_rejected_total` counter to Prometheus.
- [ ] Background sweeper goroutine.
- [ ] Update error code documentation.

## Tests

- [ ] `ratelimit_test.go`: hammer a single tenant; first N requests
      pass, then 429s appear; after 1 s, requests pass again.
- [ ] Two tenants don't interfere: tenant A is throttled, tenant B
      passes at full rate.
- [ ] Sweeper test: register 1 000 tenants, advance simulated time by 1
      hour, confirm map shrinks.
- [ ] Middleware off-by-default: confirm zero overhead when
      `--rate-limit-rps 0`.

## Verification

- [ ] Load test ([P9](09-benchmarks.md)) re-run with rate limit ON
      shows clean per-tenant cap.
- [ ] Metrics endpoint exposes the new gauges.
- [ ] `serve --help` documents the flags.

## Gotchas

1. **Burst vs sustained**. The token bucket allows short bursts up to
   `burst` tokens. Set burst ≈ 2-3× RPS for typical UI bursts.
2. **Cancelled requests still consume tokens**. Acceptable trade-off
   (otherwise abusers could DoS by opening then cancelling).
3. **Map contention**. The single mutex over `buckets` is fine up to
   ~100 k RPS; if it ever bottlenecks, switch to sharded maps.
4. **Distributed deployments**. If the server is replicated behind a
   load balancer, each replica enforces its own limit independently. For
   strict cross-replica limits, defer to the reverse-proxy or an
   external service (Redis, Envoy). Not handled here.

## PR layout

Single PR: `feat(server): per-tenant rate limiting middleware (opt-in)`.
