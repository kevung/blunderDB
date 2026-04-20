# 12 — Extract shared test utilities

**Status: Done**

**Goal:** Replace independent `setup*DB` helpers with a single shared `test_helpers_test.go` providing common fixtures and setup functions.

**Depends on:** Nothing (can be done at any time).

**Impact:** Low — reduces test boilerplate, makes writing new tests easier.

## What was done

### 1. Created `test_helpers_test.go`

Shared helpers using `t.Cleanup` instead of returning cleanup functions:

- `newTestDB(t)` — file-backed DB in `t.TempDir()`, returns `*Database`
- `newTestDBWithXG(t)` — creates DB + imports `testdata/test.xg`
- `newTestDBWithSGF(t)` — creates DB + imports `testdata/test.sgf`
- `importTestMatch(t, db)` — imports `testdata/test.sgf`, returns match ID (moved from `collection_test.go`)
- `getPositionIDs(t, db, limit)` — returns first N position IDs (moved from `collection_test.go`)
- `positionCount(t, db)` / `matchCount(t, db)` — assertion helpers

### 2. Migrated 6 test files

| File | Old helper | Replacement |
|---|---|---|
| `gnubg_import_test.go` | `setupTestDB` (definition removed) | `newTestDB` |
| `collection_test.go` | `setupCollectionTestDB` + `importTestMatch` + `getPositionIDs` (all removed) | `newTestDB` + shared helpers |
| `anki_test.go` | `setupAnkiTestDB` (removed), `setupAnkiCollectionWithPositions` (simplified) | `newTestDB` |
| `tournament_test.go` | `setupTournamentTestDB` (removed) | `newTestDB` |
| `search_rewrite_test.go` | `setupSearchTestDB` (simplified to delegate) | `newTestDBWithXG` |
| `mat_position_test.go` | All `setupTestDB` call sites | `newTestDB` |

### Not migrated (by design)

- `setupBenchDB` (benchmark_test.go) — uses `sync.Once` + walks `testdata/tournois/`, too specialized
- `setupExportTestDB` (export_test.go) — creates complex synthetic data, not a simple DB+import
- `setupCLI`/`setupCLIWithDB` (cli_test.go) — returns `*CLI`, different type

## Files touched

- **New:** `test_helpers_test.go`
- **Edit:** `gnubg_import_test.go`, `collection_test.go`, `anki_test.go`, `tournament_test.go`, `search_rewrite_test.go`, `mat_position_test.go`

## Acceptance criteria

- [x] `test_helpers_test.go` exists with shared DB setup functions
- [x] At least 3 of the 4 existing setup helpers replaced (5 replaced + 1 simplified)
- [x] All tests pass: `go test -count=1 -timeout 300s ./...`
- [x] Collection/tournament/Anki tests use shared helpers

## Rollback

`git revert` — additive file + mechanical call-site changes.
