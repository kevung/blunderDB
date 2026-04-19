# blunderDB Improvement Plan

Comprehensive analysis and prioritised improvement roadmap.
Date: 2025-04-19

---

## Executive Summary

blunderDB is a capable backgammon analysis tool with a solid domain model and
working multi-format import pipeline. However, the codebase has accumulated
significant technical debt that impacts maintainability, reliability, and
developer velocity. The three most impactful issues are:

1. **`db.go` is 16,223 lines** — a single file with 174 methods and 282 functions
2. **Zero tests run in CI** — the 11k-line test suite only runs locally
3. **Frontend god component** — `App.svelte` is 4,866 lines with 103 functions

This plan is organised into 7 focus areas with concrete, actionable tasks.

---

## Table of Contents

1. [Backend Architecture](#1-backend-architecture)
2. [Error Handling & Logging](#2-error-handling--logging)
3. [API Design](#3-api-design)
4. [Testing](#4-testing)
5. [CI/CD](#5-cicd)
6. [Frontend Architecture](#6-frontend-architecture)
7. [Documentation](#7-documentation)

---

## 1. Backend Architecture

### 1.1 Split `db.go` into domain-focused files

`db.go` (16,223 lines, 174 methods, 282 functions) is the single biggest
maintainability problem. Every concern lives in one file: schema, migrations,
CRUD, search, three import pipelines, export, collections, tournaments, Anki,
session state, MET computation, and helper utilities.

**Proposed split** (all remain `package main` for Wails binding compatibility):

| File                  | Content                                         | Est. lines |
|-----------------------|-------------------------------------------------|------------|
| `db.go`               | `Database` struct, `SetupDatabase`, PRAGMAs     |      400   |
| `db_schema.go`        | `ensureAllTablesExist`, index creation           |      500   |
| `db_migration.go`     | All `migrate_X_to_Y` functions                  |    1,200   |
| `db_position.go`      | Position CRUD, compact encoding, reconstruction |    1,000   |
| `db_analysis.go`      | Analysis CRUD, compression, merge logic         |      800   |
| `db_search.go`        | `loadPositionsByFiltersCore`, filter helpers     |    1,500   |
| `db_import_xg.go`     | `ImportXGMatch`, XG-specific helpers             |    2,000   |
| `db_import_gnubg.go`  | `ImportGnuBGMatch`, GnuBG-specific helpers       |    2,000   |
| `db_import_bgf.go`    | `ImportBGFMatch`, BGF-specific helpers            |    2,000   |
| `db_import_common.go` | Shared import cache, `savePositionInTxWithCache` |      500   |
| `db_export.go`        | `ExportDatabase`, `ExportCollections`, etc.      |    1,500   |
| `db_match.go`         | Match/game/move CRUD, `GetMatchMovePositions`    |      600   |
| `db_collection.go`    | Collection CRUD, reorder, import/export          |      600   |
| `db_tournament.go`    | Tournament CRUD, match assignment, export        |      600   |
| `db_anki.go`          | Anki deck/card CRUD, FSRS review logic           |      500   |
| `db_met.go`           | GnuBG Match Equity Table (Kazaross-XG2 + Zadeh) |      500   |
| `db_comment.go`       | Comment CRUD, search                             |      300   |
| `db_session.go`       | Session state, command/search history            |      400   |
| `db_helpers.go`       | `parseIntFilterExpr`, `roundToMillipoint`, etc.  |      300   |

**Rules:**
- Pure file split — no refactoring, no API changes.
- All files in `package main` (Wails binding requirement).
- One PR per 2-3 files to keep reviews manageable.
- Run `go test ./...` after each split to verify nothing broke.

### 1.2 Reduce import pipeline duplication

The XG, GnuBG, and BGF import pipelines share an identical high-level structure
(~21 parallel functions across 3 formats). Extract common patterns:

| Shared pattern                          | Current duplication          |
|-----------------------------------------|------------------------------|
| Date parsing with format fallback       | Copy-pasted 4 times          |
| Match hash check + canonical hash check | 3 identical sequences        |
| Transaction setup + commit/rollback     | 3 identical blocks           |
| Position normalization + Zobrist save   | Already shared, good         |
| `saveMoveAnalysisInTx` signature        | 3 near-identical functions   |

**Actions:**
- Extract `parseMatchDate(dateStr string) (time.Time, error)` helper.
- Extract `checkDuplicateMatch(hash, canonicalHash) (existingID, error)` helper.
- Keep format-specific analysis conversion functions separate (they differ enough).

### 1.3 Switch from `sync.Mutex` to `sync.RWMutex`

The single `d.mu` `sync.Mutex` serialises ALL 174 methods — reads block reads.
With SQLite WAL mode already enabled, concurrent readers are safe at the DB level.

**Action:** Change `d.mu` to `sync.RWMutex`. Replace `d.mu.Lock()` with
`d.mu.RLock()` in read-only methods (~60 methods: `Get*`, `Load*`, `Search*`).
Keep `d.mu.Lock()` for writes.

**Risk:** Low — WAL mode already handles reader/writer concurrency. The mutex
only needs to prevent concurrent writes (which WAL also handles, but the Go side
should serialise transaction opens).

---

## 2. Error Handling & Logging

### 2.1 Replace `fmt.Println` with structured logging

**Current state:** 383 `fmt.Println`/`fmt.Printf` calls in `db.go`, 358 in
`cli.go`, 4 in `main.go`. In GUI mode, stdout goes nowhere — these errors are
silently lost.

**Action:**
- Introduce `log/slog` (stdlib, Go 1.21+) with a `slog.Handler` that:
  - In CLI mode: writes to stderr with text format.
  - In GUI mode: writes to a ring buffer accessible via a "Debug Log" panel
    (or a log file at `XDG_STATE_HOME/blunderDB/blunderdb.log`).
- Replace all `fmt.Println("Error ...")` with `slog.Error("...", "err", err)`.
- Remove double-reporting (print + return err) — just return the error; let the
  caller decide whether to log.

### 2.2 Fix silent exit-zero on startup failure

`main.go` prints errors and returns (exit code 0) for fatal failures:
- Config load failure
- Database setup failure

**Action:** Replace with `log.Fatal(err)` or `os.Exit(1)` for unrecoverable
startup errors.

### 2.3 Remove diagnostic prints from production code

`main.go` lines 71-72 print window dimensions on every startup. Either gate
behind a `-v`/`--verbose` flag or remove.

### 2.4 Use consistent error wrapping

Newer code uses `fmt.Errorf("context: %w", err)` (wrapping), older code uses
`%v` or bare returns. Standardise on `%w` for all errors to enable
`errors.Is`/`errors.As` throughout.

---

## 3. API Design

### 3.1 Replace 33-parameter `LoadPositionsByFilters` with a struct

The current function signature is unmaintainable:

```go
func (d *Database) LoadPositionsByFilters(
    filter Position, includeCube bool, includeScore bool,
    pipCountFilter string, winRateFilter string, ...,
    /* 33 parameters total */
) ([]Position, error)
```

**Action:** Define a `SearchFilters` struct in `model.go`:

```go
type SearchFilters struct {
    Filter             Position
    IncludeCube        bool
    IncludeScore       bool
    PipCountFilter     string
    WinRateFilter      string
    GammonRateFilter   string
    // ... all 33 fields
}
```

Update `LoadPositionsByFilters` to `LoadPositionsByFilters(f SearchFilters)`.
Wails auto-generates TypeScript bindings for structs, so the frontend migration
is mechanical.

### 3.2 Replace 12-parameter `ExportDatabase` with an options struct

Same pattern — `ExportOptions` struct with all boolean/list fields.

### 3.3 Replace `map[string]interface{}` returns with typed structs

Several methods return untyped maps (e.g. `PositionExists` returns
`map[string]interface{}{"id": ..., "exists": ...}`). Replace with named structs
for type safety in auto-generated bindings.

---

## 4. Testing

### 4.1 Current state

| Metric              | Value       |
|---------------------|-------------|
| Test files          | 28          |
| Test functions      | 127 + 10 benchmarks |
| Total test lines    | 11,315      |

#### What's well-tested
- Position deduplication & canonical hashes
- XG and GnuBG import pipelines
- Export variations (20+ tests)
- Migration chain (v1.0→v2.3)
- EPC computation
- Bitboard masks
- Analysis compression
- Schema benchmarks

#### What's untested

| Area                            | Lines | Risk    |
|---------------------------------|------:|---------|
| CLI (`cli.go`)                  | 2,112 | **High** — user-facing entry point |
| App (`app.go`)                  |   305 | Medium — Wails-dependent |
| Config (`config.go`)            |   104 | Low     |
| Collection CRUD                 |  ~600 | Medium  |
| Tournament CRUD                 |  ~600 | Medium  |
| Anki/FSRS logic                 |  ~500 | Medium  |
| Session state                   |  ~400 | Low     |
| Comment CRUD                    |  ~300 | Low     |
| Import cancellation             |   ~20 | Medium — untested atomic flag path |
| Concurrent mutex contention     |     — | Medium  |
| Frontend (`commandProcessor.js`)| 322   | **High** — command parser, pure functions |
| Frontend (all)                  |22,381 | High — zero test infrastructure |

### 4.2 Backend test improvements

**P0 — High impact, low effort:**

1. **CLI round-trip test.** Import a match via `CLI.RunImport()`, list stats
   via `CLI.RunList()`, search via `CLI.RunSearch()`, export via
   `CLI.RunExport()`. Verifies the full CLI pipeline end-to-end.

2. **Collection CRUD tests.** Create, add positions, reorder, move between
   collections, delete. Verify FK cascades.

3. **Tournament CRUD tests.** Create, add matches, reorder, remove matches,
   delete. Verify matches are unlinked (not deleted) when tournament is removed.

4. **Anki/FSRS tests.** Create deck, sync from collection, get next card,
   review with each rating (Again/Hard/Good/Easy), verify FSRS state transitions.

5. **Import cancellation test.** Start an import in a goroutine, set
   `importCancelled = 1` after N rows, verify partial rollback.

**P1 — Medium effort:**

6. **Concurrent read/write test.** Launch N goroutines reading while one writes.
   Verify no panics, no data races (use `go test -race`).

7. **Comment CRUD tests.** Add, update, delete, search.

8. **Session state tests.** Save and restore session state round-trip.

### 4.3 Frontend test infrastructure

1. **Install vitest** + `@testing-library/svelte`:
   ```bash
   cd frontend && npm install -D vitest @testing-library/svelte jsdom
   ```

2. **Priority test targets:**
   - `commandProcessor.js` — pure function, testable without DOM. Cover all
     command variants: filter, search, collection, deck, export.
   - Store logic in `stores/` — test derived store computations.
   - `SearchModal.svelte` — verify filter state management.

3. **Add `npm test` script** to `package.json`.

### 4.4 Extract shared test utilities

Four independent `setup*DB` helpers exist across test files. Extract to a shared
`test_helpers_test.go`:

```go
func setupTestDB(t *testing.T) *Database {
    t.Helper()
    db := &Database{}
    err := db.SetupDatabase(":memory:")
    require.NoError(t, err)
    return db
}
```

---

## 5. CI/CD

### 5.1 Current state

CI builds on 4 platforms but **runs zero tests, zero linting, zero static
analysis**. The entire 11k-line test suite only runs locally.

### 5.2 Add test step to CI

```yaml
- name: Run tests
  run: go test -timeout 300s -race ./...
  shell: bash
```

Add this after the Go setup step, before the Wails build. The `-race` flag
catches data races (important given the mutex patterns).

### 5.3 Add static analysis

```yaml
- name: Go vet
  run: go vet ./...
  shell: bash

- name: golangci-lint
  uses: golangci/golangci-lint-action@v4
  with:
    version: latest
```

`golangci-lint` catches: shadow variables, unchecked errors, inefficient string
concatenation, unused code, and 50+ other issues.

### 5.4 Add frontend lint + build check

```yaml
- name: Frontend lint
  working-directory: frontend
  run: npm ci && npx svelte-check
  shell: bash
```

### 5.5 Add benchmark regression tracking (nice-to-have)

On PRs, run benchmarks and compare with `benchstat` against main. Post results
as a PR comment. Catches silent performance regressions.

---

## 6. Frontend Architecture

### 6.1 Break up `App.svelte`

`App.svelte` (4,866 lines, 103 functions, 33 store subscriptions) orchestrates
every feature. Target: reduce to <500 lines by extracting logic.

**Phase 1 — Extract service modules** (no component changes):
- `services/databaseService.js` — open, save, close, import database files
- `services/positionService.js` — load, save, navigate, mirror positions
- `services/importService.js` — XG/GnuBG/BGF import orchestration
- `services/exportService.js` — export database/collections/tournaments
- `services/clipboardService.js` — copy position/analysis to clipboard
- `services/sessionService.js` — save/restore session state

Each service is a plain JS module that calls `wailsjs` bindings and updates
stores. App.svelte becomes a thin orchestrator that wires services to the
component tree.

**Phase 2 — Extract feature containers:**
- `MatchView.svelte` — match navigation mode (currently inline in App.svelte)
- `PositionView.svelte` — standalone position viewing
- `DatabaseView.svelte` — filtered position list browsing

### 6.2 Consolidate modal visibility management

**Current:** 20+ individual `writable(false)` stores in `uiStore.js`, each
needing manual subscription in App.svelte, plus 3 derived stores
(`isAnyModalOrPanelOpenStore`, `isAnyModalOpenStore`, `isAnyPanelOpenStore`)
that manually enumerate every flag.

**Proposed:** Single `activeModal` store + `openPanels` Set:

```js
// uiStore.js
export const activeModal = writable(null);     // 'search' | 'export' | 'help' | null
export const openPanels = writable(new Set()); // Set<'match' | 'collection' | ...>

export const isAnyModalOpen = derived(activeModal, $m => $m !== null);
export const isAnyPanelOpen = derived(openPanels, $p => $p.size > 0);
```

Adding a new modal = adding one string constant. No new stores, no new
derived subscriptions.

### 6.3 Migrate to Svelte 5 runes

The app runs Svelte 5 in compatibility mode. Zero `$state`/`$derived`/`$effect`
usage. Migration yields:
- Elimination of 33 manual `subscribe`/`onDestroy` pairs in App.svelte.
- Fine-grained reactivity (better perf for large position lists).
- Simpler component APIs (`$props()` vs `export let`).

**Strategy:** Migrate one component at a time, starting with leaf components
(StatusBar, WarningModal) and working inward. Stores can coexist — Svelte 5
runes work alongside `writable` during transition.

### 6.4 Parameterise duplicated table components

Six near-identical table modal components (~98 lines each):
- `TakePoint2LastModal.svelte`, `TakePoint2LiveModal.svelte`
- `TakePoint4LastModal.svelte`, `TakePoint4LiveModal.svelte`
- `GammonValue1Modal.svelte`, `GammonValue2Modal.svelte`

Replace with a single `TableModal.svelte` parameterised by data source and title.

### 6.5 Add frontend linting

No linter or formatter is configured.

```bash
npm install -D eslint prettier eslint-plugin-svelte
```

Add `eslint.config.js` and `prettier.config.js`. Alternatively, use
[Biome](https://biomejs.dev/) as a single tool for both linting and formatting.

### 6.6 Strip console.log from production

156 `console.*` calls in App.svelte alone. Options:
- Use a `DEBUG` env var to gate console output.
- Use a Vite plugin to strip console calls in production builds.
- Replace with a lightweight logger that's silent in production.

### 6.7 Improve accessibility

- Add `role="dialog"` + `aria-modal="true"` to all modal components.
- Add focus trapping to modals (trap Tab key within modal while open).
- Add `aria-label` to icon-only buttons in Toolbar.
- Fix the resize handle: add `role="separator"` + `aria-orientation`.
- Audit keyboard navigation for all interactive features.

---

## 7. Documentation

### 7.1 Improve README.md

Current README is a single badge + link. Add:
- Feature overview (one paragraph)
- Screenshot
- Build instructions (Go + Node prerequisites, `wails dev`, `wails build`)
- CLI quick-start example
- Link to full docs

### 7.2 Document undocumented CLI commands

`main.go` dispatches 12 commands but `CLI_USAGE.md` only documents 7. Missing:
`create`, `match`, `verify`, `info`, `edit`. Either document them or mark as
internal/experimental.

### 7.3 Fix `config.yaml` naming confusion

`config.go` saves JSON to a file named `config.yaml`. Either:
- Rename to `config.json`, or
- Switch to actual YAML marshalling (adds a dependency).

Renaming is simpler and doesn't break existing installs if a migration reads
both paths.

### 7.4 Clean up stale plan documents

9 historical markdown files in the repo root (`ANALYSIS_IMPLEMENTATION.md`,
`MATCH_IMPORT_ARCHITECTURE.md`, `POSITION_TRACKING_IMPLEMENTATION.md`, etc.).
Move to a `docs/design/` directory to reduce root clutter, or delete those that
are fully superseded by the current code.

### 7.5 Close open FOLLOWUPS

Per `FOLLOWUPS.md`:
1. `wails build` smoke test — add to CI or document as manual step.
2. GUI manual smoke test on migrated production DB — manual, document procedure.
3. `BenchmarkSearch_WinGammonCombo` TEMP B-TREE bottleneck — investigate
   composite index or query restructuring.

---

## Implementation Priority

### Phase 1 — Safety net (1-2 weeks)

| # | Task                                   | Impact | Effort |
|---|----------------------------------------|--------|--------|
| 1 | Add `go test -race ./...` to CI        | **Critical** | Low |
| 2 | Add `go vet` + `golangci-lint` to CI   | High   | Low    |
| 3 | Fix exit-zero on startup failure       | Medium | Trivial|
| 4 | Add CLI round-trip integration test    | High   | Medium |
| 5 | Add collection/tournament/Anki tests   | Medium | Medium |

### Phase 2 — Backend cleanup (2-3 weeks)

| # | Task                                     | Impact | Effort |
|---|------------------------------------------|--------|--------|
| 6 | Split `db.go` into ~18 files             | **Critical** | Medium |
| 7 | Introduce `slog` logging                 | High   | Medium |
| 8 | Extract `SearchFilters` struct           | High   | Medium |
| 9 | Extract `ExportOptions` struct           | Medium | Low    |
| 10| Switch to `sync.RWMutex`                 | Medium | Low    |
| 11| Extract shared import helpers            | Medium | Medium |
| 12| Extract shared test utilities            | Low    | Low    |

### Phase 3 — Frontend modernisation (3-4 weeks)

| # | Task                                      | Impact | Effort |
|---|-------------------------------------------|--------|--------|
| 13| Extract service modules from App.svelte   | **Critical** | High |
| 14| Consolidate modal store management        | High   | Medium |
| 15| Install vitest + test commandProcessor.js | High   | Medium |
| 16| Add eslint/prettier to frontend           | Medium | Low    |
| 17| Strip/gate console.log calls              | Low    | Low    |
| 18| Parameterise duplicated table modals      | Low    | Low    |

### Phase 4 — Polish (ongoing)

| # | Task                                | Impact | Effort |
|---|-------------------------------------|--------|--------|
| 19| Svelte 5 runes migration            | Medium | High   |
| 20| Accessibility improvements          | Medium | Medium |
| 21| Improve README.md                   | Medium | Low    |
| 22| Document missing CLI commands       | Low    | Low    |
| 23| Clean up stale design docs          | Low    | Low    |
| 24| Benchmark regression tracking in CI | Low    | Medium |

---

## Metrics

Track these to measure progress:

| Metric                          | Current       | Target          |
|---------------------------------|---------------|-----------------|
| `db.go` line count              | 16,223        | <500            |
| Largest Go file                 | 16,223 (db.go)| <2,500          |
| Tests in CI                     | 0             | All             |
| `fmt.Println` in Go source     | 745           | 0               |
| `App.svelte` line count        | 4,866         | <500            |
| Frontend test count             | 0             | 50+             |
| `console.*` in App.svelte      | 156           | 0 in production |
| `LoadPositionsByFilters` params | 33            | 1 (struct)      |
