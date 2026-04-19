# 00 — Baseline benchmarks

**Goal:** Record current-code timings for batch import and filtered search so v2.0.0 gains are provable. Numbers become unrecoverable once the schema changes — do this first.

**Depends on:** nothing.

**Does NOT touch:** schema, imports, search. Only adds a test file.

## Files touched

- **New:** `benchmark_test.go` at repo root (same `package main`).
- **New:** `testdata/tournois.db` (gitignored — rebuilt by the harness).

## Sub-tasks

### 1. Build the fixture harness

- [ ] In `benchmark_test.go`, add a `TestMain` / `sync.Once` that:
  - Walks `testdata/tournois/` for `.xg`, `.sgf`, `.mat`, `.bgf` files.
  - Calls `(*Database).ImportXGMatch` / `ImportGnuBGMatch` / `ImportBGFMatch` based on extension, exactly as `cli.go:importBatch` does (db.go:5652 / 8224 / 9936).
  - Targets an **in-memory** DB (`:memory:`) by default, or `testdata/tournois.db` when `BENCH_DISK=1` is set.
  - Reuses the imported DB across every `Benchmark*` in the file (don't reimport per benchmark).
- [ ] Add `testdata/tournois.db` and `*.db-wal` / `*.db-shm` to `.gitignore` if not already.

### 2. Benchmark functions (current code)

Each benchmark calls into existing `Database` methods — no new helpers.

- [ ] `BenchmarkImport_TournoisCold` — fresh `:memory:` DB, import every tournament file. Measures the end-to-end batch path.
- [ ] `BenchmarkImport_TournoisIncremental` — import the smallest match into an already-full DB. Isolates the "load all positions into RAM" warmup cost (db.go:5805–5837).
- [ ] `BenchmarkSearch_DecisionCube` — `LoadPositionsByFilters` with `decisionTypeFilter=true` and no other filter.
- [ ] `BenchmarkSearch_ErrorAboveTenth` — `errorMin=0.1` (cube + move error).
- [ ] `BenchmarkSearch_PipWindow` — pip-count filter string `-2..2`.
- [ ] `BenchmarkSearch_WinGammonCombo` — player1 win >55% AND gammon >20%.
- [ ] `BenchmarkSearch_ScoreSpecific` — `match_length=7`, `score_1=6`, `score_2=4`.
- [ ] `BenchmarkSearch_DiceAndPlayer` — `dice=(6,5)` AND `player_on_roll=0`.
- [ ] `BenchmarkSearch_CheckerStructure` — a small "anchor on 20-point" template Position.
- [ ] `BenchmarkSearch_PrimePattern` — a 5-prime template (will become the bitboard showcase in phase 05).

### 3. Baseline capture

- [ ] Run `go test -bench=. -benchtime=5x -run=^$ ./... | tee bench-before-v2.txt`.
- [ ] Copy `bench-before-v2.txt` into the repo at `tasks/bench-before-v2.txt`. **Commit it** — future comparison depends on it.
- [ ] Also note the absolute wall-clock of a human-visible import:
  ```bash
  rm -f /tmp/tournois.db
  time find testdata/tournois -type f \( -name '*.xg' -o -name '*.sgf' -o -name '*.mat' -o -name '*.bgf' \) -print0 \
    | xargs -0 -n1 -I{} ./blunderdb import --db /tmp/tournois.db --type match --file {}
  ```
  Record the `real` time in the commit message.

## Acceptance criteria

- [ ] `go test -bench=. -run=^$ ./...` runs cleanly on current `main` code with no edits to non-test files.
- [ ] `bench-before-v2.txt` exists under `tasks/` and is committed.
- [ ] The fixture-builder helper is reusable from later sheets (01–06 can all call it).

## Risks

- **Test file in `package main`.** `benchmark_test.go` lives at repo root in the same package as `main.go` / `db.go`. Don't name any helper `main` or collide with existing test names (check `export_test.go`, `migration_test.go`).
- **Fixture size drift.** If `testdata/tournois/` gains/loses files between runs, numbers become noisy. Print the file count + total bytes at the start of `TestMain` so a re-run's output self-documents.
- **Random benchmark order.** `go test -bench=.` iterates alphabetically — benchmark names above are already ordered so reading the output tracks the plan. Keep it that way.
