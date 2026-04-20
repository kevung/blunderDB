# 24 — Benchmark regression tracking in CI

**Goal:** Run Go benchmarks in CI and track results over time to catch performance regressions in critical paths (search, import, EPC).

**Depends on:** 01 (CI running tests) — benchmarks extend the CI pipeline.

**Impact:** Medium — prevents silent performance regressions, especially in the search and import hot paths.

## Context

- 10 benchmarks already exist across the test suite (`benchmark_test.go`, `schema_benchmark_test.go`)
- CI currently runs zero benchmarks
- Known bottleneck: `WinGammonCombo` (referenced in `FOLLOWUPS.md`)
- Schema v2.0.0 added denormalized filter columns and bitboard indexes for search performance — regressions here would be costly

## Files touched

- **Edit:** `.github/workflows/build.yml` — add benchmark step
- **New:** `.github/workflows/benchmark.yml` (optional: separate workflow)
- **New:** `scripts/bench-compare.sh` (optional: compare benchmark results)

## Tasks

### 1. Inventory existing benchmarks

- [x] List all benchmarks: 10 existing (2 import, 8 search — all require `testdata/tournois/`)
- [x] Verify they all pass locally: all 10 pass
- [x] Note any flaky or slow benchmarks: `BenchmarkSearch_WinGammonCombo` (~409ms) is the slowest

### 2. Add benchmark step to CI

- [x] Add `benchmark` job to `.github/workflows/build.yml`
- [x] Runs `go test -bench=. -benchmem -count=3 -timeout=10m`
- [x] Uploads `bench.txt` as artifact

### 3. Add benchmark comparison (optional)

- [ ] Deferred — benchstat comparison can be added later when baseline is established in CI

### 4. Add critical benchmark markers

- [x] Added 7 CI-friendly benchmarks using committed fixtures or pure computation:
  - `BenchmarkEPC` — EPC bearoff calculation (5 board configs)
  - `BenchmarkZobristHash` — Zobrist hash for 2 typical positions
  - `BenchmarkOccupancyMasks` — bitboard mask computation
  - `BenchmarkPipCounts` — pip count computation
  - `BenchmarkImport_SingleXG` — import of committed `testdata/test.xg`
  - `BenchmarkImport_SingleSGF` — import of committed `testdata/test.sgf`
  - `BenchmarkSearch_SmallDB` — search on small dataset from committed fixtures
- [x] Changed `setupBenchDB` to `Skip` (not `Fatal`) when `testdata/tournois/` is missing — CI-safe

### 5. Set performance budgets (optional)

- [ ] Deferred — establish baselines from CI first

### 6. Verify

- [x] All 17 benchmarks pass locally (10 tournois + 7 CI-friendly)
- [x] CI pipeline includes benchmark job
- [x] Results uploaded as artifact

## Acceptance criteria

- [x] Benchmarks run in CI on every push/PR
- [x] Results are saved as artifacts (downloadable)
- [x] 7 CI-friendly benchmarks always run; 10 tournois benchmarks skip gracefully in CI
- [ ] Optional: benchstat comparison shows no regressions vs. baseline (deferred)

## Rollback

`git revert` — CI config only, no code changes.
