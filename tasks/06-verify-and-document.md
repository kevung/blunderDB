# 06 — Verify & document

**Goal:** Prove the v2.0.0 gains with real numbers, run the full regression surface, update `CLAUDE.md`, prepare the release.

**Depends on:** 04 + 05 both merged.

## Files touched

- `CLAUDE.md` — add the 500-line doc rule + v2.0.0 schema paragraph.
- `tasks/bench-after-v2.txt` — committed numbers.
- `tasks/search-query-plans.txt` — committed `EXPLAIN QUERY PLAN` dumps (from phase 05).
- `doc/source/index.rst` — changelog entry (via `scripts/release.sh --changelog`).

## 1. Capture post-change benchmark numbers

- [ ] On a clean tree (all of 01–05 merged), run:
  ```bash
  go test -bench=. -benchtime=5x -run=^$ ./... | tee tasks/bench-after-v2.txt
  ```
- [ ] Compare to `tasks/bench-before-v2.txt` (sheet 00). For each benchmark, compute the ratio; note any benchmark that did not improve and investigate.
- [ ] Commit `bench-after-v2.txt`.

## 2. Full regression run

- [ ] `go test ./...`
- [ ] `go test ./tests/...`
- [ ] `go vet ./...`
- [ ] Build the binary: `wails build`
- [ ] Build with the alternate tag: `wails build -tags webkit2_41` (only if you're on Linux; CI covers this otherwise).

## 3. Real-database migration smoke

- [ ] Copy a real 1.8.0 production-size DB to `/tmp/pre-migration.db` (do **not** commit it — `.gitignore` covers `*.db`).
- [ ] Run `./blunderdb list --db /tmp/pre-migration.db --type stats` to trigger migration. Time it.
- [ ] Spot-check: `./blunderdb search --db /tmp/pre-migration.db --decision cube` vs what you remember it returning. Same count?
- [ ] Open in the GUI, click through every filter in the Search modal. Every filter returns results without errors.

## 4. `CLAUDE.md` updates

Append two short paragraphs to the "Notes & Gotchas" section or equivalent:

- [ ] **Doc size rule.** "Documents I generate (plans, task sheets, design notes) must stay ≤500 lines each. Split long docs into a README index + per-topic files."
- [ ] **v2.0.0 schema shape.** One paragraph naming the key ideas: Zobrist hash column + unique index on `position`, denormalized filter columns on `position` and `analysis`, bitboard occupancy/point masks for pattern pre-filter, WAL + tuned PRAGMAs. Point readers at `DATABASE_OPTIMIZATION_PLAN.md` and `tasks/`.
- [ ] Confirm `CLAUDE.md` itself is still under the 500-line limit after the additions.

## 5. Changelog + version

- [ ] Decide the release version. Options:
  - **0.16.0** if strictly following the existing metaStore-driven numbering (`frontend/src/stores/metaStore.js`).
  - **1.0.0** if you want to mark v2.0.0 DB schema as a milestone.
  - Recommendation: **0.16.0 app + DB schema v2.0.0**. The app version in metaStore and the DB schema version are independent.
- [ ] Run `./scripts/release.sh 0.16.0 --changelog "Database schema v2.0.0: Zobrist-hashed position dedup, denormalized filter columns, bitboard pattern pre-filter, WAL journaling. Batch import ≥3× faster, filtered search ≤100 ms on 10k+ positions."`
- [ ] Review the generated commit + tag before pushing. `--push` is a separate flag; don't pass it in this sheet — push is a judgment call for the maintainer.

## 6. Release-notes draft

- [ ] In the changelog entry, call out:
  - **Breaking**: DB files created with v2.0.0 cannot be opened by pre-v2.0.0 binaries.
  - **Auto-migration**: opening an old DB in v2.0.0 rewrites it in place; back up first.
  - **Performance**: post-change numbers from `bench-after-v2.txt`.

## 7. Close the task sheets

- [ ] Update `tasks/README.md`'s status table — all boxes checked.
- [ ] If any sub-task in 00–05 was skipped or deferred, open a follow-up in `tasks/FOLLOWUPS.md` with a one-line description. Don't leave silent gaps.

## Acceptance criteria

- [ ] `bench-after-v2.txt` shows ≥3× import speedup and ≤100 ms per search benchmark.
- [ ] Full test suite green on a local `wails build`.
- [ ] GUI exercised manually on a real migrated DB; no filter returns an empty set that previously returned results.
- [ ] `CLAUDE.md` is updated and still ≤500 lines.
- [ ] Tag `v0.16.0` (or chosen version) created locally, ready to push.

## Risks

- **Benchmark drift on dev machine.** Run benchmarks with `-benchtime=5x` and on AC power, not battery. If numbers disagree wildly between runs, bump to `-benchtime=30x` for the ones you commit.
- **Frontend bindings out of date.** If phase 04 or 05 added any exported method on `Database`, restart `wails dev` before the manual smoke test so `frontend/wailsjs/go/main/Database.js` regenerates.
- **Forgetting the migration warning in release notes.** Users with automation that hot-copies the DB while it's open will corrupt it under WAL. Mention it explicitly.
- **Version-number confusion.** DatabaseVersion "2.0.0" vs app version "0.16.0" — document both in the release notes so support questions ("why does my DB say 2.0.0 when the app is 0.16?") have an answer.
