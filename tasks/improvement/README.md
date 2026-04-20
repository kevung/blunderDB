# blunderDB Improvement Plan — Task Sheets

Detailed task sheets for every item in `IMPROVEMENT_PLAN.md`.
Each sheet is a self-contained execution unit with checkboxes, file anchors,
acceptance criteria, and rollback notes.

## Phases

Dependencies flow top-down within each phase. Phases are sequential.

### Phase 1 — Safety net

| # | Sheet | Purpose | Depends on |
|---|---|---|---|
| 01 | [ci-tests](01-ci-tests.md) | Add `go test -race` to CI | — |
| 02 | [ci-lint](02-ci-lint.md) | Add `go vet` + `golangci-lint` to CI | 01 |
| 03 | [fix-exit-zero](03-fix-exit-zero.md) | Fix silent exit-zero on startup failures | — |
| 04 | [cli-tests](04-cli-tests.md) | CLI round-trip integration tests | — |
| 05 | [crud-tests](05-crud-tests.md) | Collection / tournament / Anki tests | — |

### Phase 2 — Backend cleanup

| # | Sheet | Purpose | Depends on |
|---|---|---|---|
| 06 | [split-db-go](06-split-db-go.md) | Split `db.go` into ~18 domain files | Phase 1 |
| 07 | [slog-logging](07-slog-logging.md) | Replace `fmt.Println` with `slog` | 06 |
| 08 | [search-filters-struct](08-search-filters-struct.md) | Extract `SearchFilters` struct | 06 |
| 09 | [export-options-struct](09-export-options-struct.md) | Extract `ExportOptions` struct | 06 |
| 10 | [rwmutex](10-rwmutex.md) | Switch `sync.Mutex` → `sync.RWMutex` | 06 |
| 11 | [import-helpers](11-import-helpers.md) | Extract shared import helpers | 06 |
| 12 | [test-utilities](12-test-utilities.md) | Extract shared test helpers | — |

### Phase 3 — Frontend modernisation

| # | Sheet | Purpose | Depends on |
|---|---|---|---|
| 13 | [app-svelte-services](13-app-svelte-services.md) | Extract service modules from App.svelte | — |
| 14 | [modal-stores](14-modal-stores.md) | Consolidate modal visibility management | 13 |
| 15 | [frontend-tests](15-frontend-tests.md) | Install vitest + test commandProcessor.js | — |
| 16 | [frontend-lint](16-frontend-lint.md) | Add eslint/prettier | — |
| 17 | [strip-console](17-strip-console.md) | Strip/gate console.log calls | 16 |
| 18 | [table-modals](18-table-modals.md) | Parameterise duplicated table modals | 13 |

### Phase 4 — Polish

| # | Sheet | Purpose | Depends on |
|---|---|---|---|
| 19 | [svelte5-runes](19-svelte5-runes.md) | Svelte 5 runes migration | 13 + 14 |
| 20 | [accessibility](20-accessibility.md) | Accessibility improvements | 13 |
| 21 | [readme](21-readme.md) | Improve README.md | — |
| 22 | [cli-docs](22-cli-docs.md) | Document missing CLI commands | — |
| 23 | [stale-docs](23-stale-docs.md) | Clean up stale design documents | — |
| 24 | [ci-benchmarks](24-ci-benchmarks.md) | Benchmark regression tracking in CI | 01 |

## Working rules

- **Each sheet stays ≤500 lines.** Split if it grows past ~450.
- **One sheet = one branch = one PR.** Don't interleave sheets.
- **Run `go test ./...`** (backend) or `npm test` (frontend) after each sheet.
- **Update checkboxes** as you complete items.

## Current status

- [x] 01 — CI tests
- [x] 02 — CI lint
- [x] 03 — Fix exit-zero
- [x] 04 — CLI tests
- [x] 05 — CRUD tests
- [ ] 06 — Split db.go
- [ ] 07 — slog logging
- [ ] 08 — SearchFilters struct
- [ ] 09 — ExportOptions struct
- [ ] 10 — RWMutex
- [ ] 11 — Import helpers
- [ ] 12 — Test utilities
- [ ] 13 — App.svelte services
- [ ] 14 — Modal stores
- [ ] 15 — Frontend tests
- [ ] 16 — Frontend lint
- [ ] 17 — Strip console
- [ ] 18 — Table modals
- [ ] 19 — Svelte 5 runes
- [ ] 20 — Accessibility
- [ ] 21 — README
- [ ] 22 — CLI docs
- [ ] 23 — Stale docs
- [ ] 24 — CI benchmarks
