# blunderDB v2.0.0 Optimization — Task Sheets

Hierarchical breakdown of `DATABASE_OPTIMIZATION_PLAN.md`. Each sheet is an independent execution unit, sized ≤500 lines, with checkboxes keyed to concrete `file:line` anchors.

## Execution order

Dependencies flow top-down; a phase may only start after the one above is green.

| # | Sheet | Purpose | Depends on |
|---|---|---|---|
| 00 | [baseline-benchmarks.md](00-baseline-benchmarks.md) | Record current-code timings as a reference point | — |
| 01 | [foundation-utilities.md](01-foundation-utilities.md) | Pure helpers: `zobrist.go`, `bitboards.go`, column populators | 00 |
| 02 | [schema-and-pragmas.md](02-schema-and-pragmas.md) | New schema for fresh DBs + PRAGMA tuning | 01 |
| 03 | [migration-1.8-to-2.0.md](03-migration-1.8-to-2.0.md) | In-place migration of existing databases | 02 |
| 04 | [import-rewrite.md](04-import-rewrite.md) | Remove in-RAM preload, add prepared statements | 03 |
| 05 | [search-rewrite.md](05-search-rewrite.md) | Push filters to SQL, bitboard pre-filter | 03 |
| 06 | [verify-and-document.md](06-verify-and-document.md) | Benchmarks, regression, docs, release | 04 + 05 |

## Working rules

- **Each sheet stays ≤500 lines.** If a sheet grows past ~450, split it.
- **One sheet = one branch-worth of work = one PR.** Don't interleave phases.
- **Finish 00 before touching any other phase.** Baseline numbers are unrecoverable once the schema changes.
- **Re-run `go test ./... && go test ./tests/...`** at the end of every sheet — no sheet is "done" with a red suite.
- **Update the check status** in each sheet as you go so a resumed session knows where you are.
- **Reference anchors** (`db.go:1845`, `model.go:36` …) come from the plan; they may drift as edits land. When a drift matters, update the anchor in the sheet, not just in code.

## Rollback strategy

- **00–02**: additive only. Revert = `git revert`.
- **03**: migration adds columns to existing DBs. If a backfill bug ships, patching is forward-only — add a `db.go` v2.0.0 → v2.0.1 step that re-runs the affected backfill. Do NOT try to drop columns.
- **04–05**: behavior changes only. Revert = `git revert`; DB shape unaffected.
- **06**: docs + release. Trivially revertable.

## Current status

- [x] 00 — baseline benchmarks
- [x] 01 — foundation utilities
- [x] 02 — schema & PRAGMAs
- [x] 03 — migration
- [x] 04 — import rewrite
- [ ] 05 — search rewrite
- [ ] 06 — verify & document
