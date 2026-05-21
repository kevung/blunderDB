# P9 — Benchmarks & load tests

**Goal.** Validate that the engine sustains the target load (≥ 10 k RPS on
PostgreSQL, ≥ 1 k RPS on SQLite-server) and provide a regression baseline
for future changes.

**Estimate.** 3-4 days. **Risk.** Low. **PRs.** 1-2.

**Prerequisites.** [P5](05-remove-global-mutex.md), [P6](06-serve-http.md).

## Three levels of measurement

### 1. Go microbenchmarks (`go test -bench`)

```
pkg/blunderdb/storage/sqlite/bench_test.go
pkg/blunderdb/storage/postgres/bench_test.go
```

Benchmarks to write:
- `BenchmarkSavePosition` — single-row insert latency.
- `BenchmarkSavePositionBatch` — batch of 1000 in one tx.
- `BenchmarkLoadPosition` — point lookup by ID.
- `BenchmarkSearchByFilter` — typical filter from production traces.
- `BenchmarkSearchByZobrist` — hash lookup (index hit).
- `BenchmarkStatsCompute` — large aggregate (the `ComputeStats` path is
  the heaviest read path).
- `BenchmarkConcurrentInsert` — `b.RunParallel` with N goroutines, varied
  via `b.N`.
- `BenchmarkConcurrentReadWrite` — mixed workload.
- `BenchmarkImportXG` — one full XG file.

Run on both backends. Recorded in CI on each PR; regression alert if
p50 latency degrades by > 20 %.

### 2. Load test tool

```
cmd/blunderdb-loadtest/
  main.go
  scenario.go
  report.go
```

No external dependency (`wrk`, `k6`, `vegeta`) — pure Go. Reasons:
keeps the repo self-contained, builds on every supported OS, integrates
into CI easily.

Usage:
```
blunderdb-loadtest \
  --target http://localhost:8080 \
  --tenants 10000 \
  --rps 10000 \
  --duration 60s \
  --scenario mixed \
  --output report.json
```

Scenarios:
- `mixed` — 80 % reads (search/list/stats), 20 % writes (save/update).
- `import` — random tenants uploading XG files from a pool.
- `read-heavy` — 95 % reads.

Report (`report.json`):
```json
{
  "duration_s": 60,
  "total_requests": 599842,
  "rps_achieved": 9997.4,
  "errors": 12,
  "latency_ms": {"p50": 18, "p95": 47, "p99": 142, "max": 410},
  "by_endpoint": { … }
}
```

Plain-text companion (`report.md`) for human review.

### 3. Scenario: 10 k concurrent tenants

The realistic shape: not 10 k open connections, but 10 k distinct tenants
each issuing ~1 req/s. Postgres `MaxConns = 50` handles this comfortably
because requests are short.

Validation criteria (proposed; refine empirically):

| Metric | SQLite-server | Postgres |
|---|---|---|
| Sustained RPS | ≥ 1 000 | ≥ 10 000 |
| p50 latency | ≤ 10 ms | ≤ 20 ms |
| p95 latency | ≤ 30 ms | ≤ 50 ms |
| p99 latency | ≤ 100 ms | ≤ 200 ms |
| Error rate | ≤ 0.01 % | ≤ 0.01 % |

If targets are not met, identify the bottleneck (CPU, disk, pool
saturation, lock contention) and either tune or document the limit in
`perf-baseline.md`.

## Hardware baseline

Document the test rig in `perf-baseline.md`:
- CPU model, core count
- RAM
- Disk type (NVMe SSD strongly recommended)
- OS / kernel
- Postgres version + relevant config (`shared_buffers`,
  `effective_cache_size`, `work_mem`, `max_connections`)
- pool size, `MaxConns`, `busy_timeout`

Two reference rigs:
- **Developer laptop**: 8-core, 32 GB, NVMe SSD.
- **Reference server**: 16-core, 64 GB, NVMe SSD, Postgres on the same host.

## Steps

- [ ] Write the microbenchmarks. Run `go test -bench=. -benchmem
      ./pkg/blunderdb/storage/...` on each backend.
- [ ] Implement `cmd/blunderdb-loadtest/`. Self-contained, single binary.
- [ ] Run the 10 k tenant scenario against Postgres (testcontainers or a
      local Postgres). Record results in `perf-baseline.md`.
- [ ] Run the same against SQLite-server to document its lower ceiling.
- [ ] Add a `make bench` Makefile target (or `scripts/bench.sh`) that
      regenerates the report.
- [ ] Wire microbenchmarks into CI as a non-blocking informational job
      (allow drift, flag regressions).

## Gotchas

1. **Postgres warm cache**. First run is cold. Pre-warm with a dummy
   scenario before measuring.
2. **Goroutine ramp-up**. Open all 10 k pseudo-clients gradually (over
   ~5 s) to avoid a thundering herd. Document `--ramp-up` flag.
3. **HTTP keep-alive**. Re-use connections (`http.Client` with default
   transport). Without keep-alive, results are dominated by TCP setup.
4. **Bench data realism**. Use fixtures that resemble production: a mix
   of cube and checker positions, varied filter complexity. Pre-seed each
   tenant with ~100 positions before the read-heavy scenarios.
5. **CI caveat**. Don't run the 10 k scenario in CI — too long, too noisy.
   CI runs microbenchmarks only. The full scenario is a manual run before
   release.

## Verification

- [ ] `make bench` produces `perf-baseline.md` with both backends
      measured.
- [ ] Postgres scenario shows ≥ 10 k RPS with p95 < 50 ms on the
      reference server. If not, document the bottleneck.
- [ ] CI microbenchmark job runs on every PR.

## Deliverables

- `cmd/blunderdb-loadtest/` binary.
- `pkg/blunderdb/storage/{sqlite,postgres}/bench_test.go`.
- `tasks/headless/perf-baseline.md` with empirical numbers.
- (optional) `Makefile` or `scripts/bench.sh`.

## PR layout

1. `bench: microbenchmarks for SQLite and Postgres backends`.
2. `bench: blunderdb-loadtest tool and perf baseline`.
