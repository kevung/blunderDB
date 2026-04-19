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

- [ ] List all benchmarks:
  ```bash
  go test -list 'Benchmark.*' ./... 2>&1 | grep '^Benchmark'
  ```
- [ ] Verify they all pass:
  ```bash
  go test -bench=. -benchtime=1x ./...
  ```
- [ ] Note any flaky or slow benchmarks

### 2. Add benchmark step to CI

- [ ] Add a benchmark job to `.github/workflows/build.yml` (or a separate `benchmark.yml`):
  ```yaml
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23.1'
      - name: Run benchmarks
        run: go test -bench=. -benchmem -count=3 -timeout=10m ./... | tee bench.txt
      - name: Upload benchmark results
        uses: actions/upload-artifact@v4
        with:
          name: benchmark-results
          path: bench.txt
  ```

### 3. Add benchmark comparison (optional)

- [ ] Use `benchstat` to compare results:
  ```yaml
  - name: Install benchstat
    run: go install golang.org/x/perf/cmd/benchstat@latest
  - name: Compare with baseline
    run: |
        # Download previous benchmark from artifact or use checked-in baseline
        benchstat baseline.txt bench.txt
  ```
- [ ] Or use a GitHub Action like `benchmark-action/github-action-benchmark` for tracking over time

### 4. Add critical benchmark markers

- [ ] Ensure benchmarks cover the critical hot paths:
  - Search with filters (already in `schema_benchmark_test.go`)
  - Position import (XG, GnuBG)
  - EPC calculation
  - Zobrist hash computation
  - Position canonical hash
- [ ] Add missing benchmarks if needed

### 5. Set performance budgets (optional)

- [ ] Define acceptable thresholds for key benchmarks:
  ```go
  // In benchmark_test.go, add a comment:
  // Budget: BenchmarkSearch should complete in < 50ms for 10k positions
  ```
- [ ] CI can fail if a benchmark regresses by > 20% (using benchstat's statistical comparison)

### 6. Verify

- [ ] CI pipeline runs benchmarks successfully
- [ ] Benchmark results are uploaded as artifacts
- [ ] Results are comparable across runs

## Acceptance criteria

- [ ] Benchmarks run in CI on every push/PR
- [ ] Results are saved as artifacts (downloadable)
- [ ] Existing 10 benchmarks all pass in CI
- [ ] Optional: benchstat comparison shows no regressions vs. baseline

## Rollback

`git revert` — CI config only, no code changes.
