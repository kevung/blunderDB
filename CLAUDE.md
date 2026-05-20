# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

blunderDB is a backgammon blunder analysis tool. It is a **Wails v2 desktop application** (Go backend + Svelte 5 / Vite frontend) distributed as a single binary that runs in either GUI or CLI mode depending on the invocation.

## Build & Run

All commands run from the repo root unless stated.

```bash
# Run the GUI in dev mode (hot-reload frontend via Vite)
wails dev

# Production build → build/bin/blunderdb
wails build

# Linux/Windows need the webkit2_41 tag to match webkit2gtk-4.1
wails build -tags webkit2_41

# Frontend alone (from frontend/)
cd frontend && npm install && npm run dev     # or: npm run build
```

CI (`.github/workflows/build.yml`) builds on ubuntu-latest (webkit2gtk-4.1), ubuntu-22.04 (webkit2gtk-4.0, no tag), windows-latest and macos-latest (`darwin/universal`). Go 1.25.0, Node v23.4.0, Wails v2.10.1.

## Running the binary

The same binary dispatches on `os.Args[1]` (see `main.go`):
- No args → GUI mode (Wails)
- First arg is `create|import|export|list|match|verify|delete|help|version|info|edit|search` → headless CLI

Full CLI reference lives in `CLI_USAGE.md`. Example:
```bash
./blunderdb import --db mymatches.db --type match --file game.xg
./blunderdb list --db mymatches.db --type stats
```

## Tests

```bash
go test ./...                     # all Go tests
go test ./tests/...               # the secondary test package
go test -run TestNameRegex ./...  # single test
```

Tests live both at the repo root (same `main` package as the app — e.g. `migration_test.go`, `export_test.go`, `gnubg_import_test.go`) and in `./tests/` (separate package for integration-style tests using fixtures in `testdata/`). Many tests rely on fixture files under `testdata/`, `gnubg/`, and on the embedded `gnubg_os6.bd` bearoff database.

## Documentation

Sphinx docs live under `doc/` (French + English). Build with `cd doc && python build.py` (requires `doc/requirements.txt` and LaTeX for the PDF build). GitHub Pages publishes from `gh-pages` on tag pushes.

## Release Process

Use `scripts/release.sh <version>` — it updates the version string in three places (`doc/source/conf.py`, `frontend/src/stores/metaStore.js`, optionally `doc/source/index.rst` changelog) and creates a commit + tag. Pushing the tag triggers the CI matrix build and publishes binaries/PDFs as a GitHub release. The `DatabaseVersion` constant in `model.go` is independent — bump it only when the SQLite schema changes (migrations live in `db.go`).

## Architecture

### Backend (Go, package `main`)

All backend Go files are in the repo root in a single `main` package:

- `main.go` — entry point; dispatches CLI vs GUI, wires Wails `Bind` for `App`, `Database`, `Config`.
- `app.go` — `App` struct, bound to the frontend. Exposes file/directory dialogs, drag-drop file reading, clipboard (xclip/wl-copy on Linux, osascript on macOS, PowerShell on Windows), and alerts via Wails runtime.
- `db.go` — `Database` struct and connection lifecycle (`NewDatabase`, `SetupDatabase`, `OpenDatabase`, `applyPragmas`). Uses `modernc.org/sqlite` (pure-Go SQLite, no CGO). In GUI mode the DB is opened in-memory (`:memory:`) and loaded from / saved to a user-chosen `.db` file via the dialogs in `app.go`. The actual domain logic is split into the sibling `db_*.go` files:
  - `db_schema.go` — table/index DDL (`ensureAllTablesExist`).
  - `db_migration.go` — `CheckVersion` and per-version upgrade paths.
  - `db_position.go` — position storage, canonical-hash dedup, scalar-column denormalization.
  - `db_analysis.go` — analysis storage and compressed equity blobs.
  - `db_match.go`, `db_tournament.go`, `db_collection.go`, `db_comment.go`, `db_anki.go`, `db_session.go`, `db_met.go` — domain-specific persistence.
  - `db_search.go`, `db_filter_match.go`, `db_stats.go` — query, filter, and aggregate logic.
  - `db_import_*.go` and `db_export.go` — import/export pipelines for XG, GnuBG, BGF, native `.db`, and JSON.
- `cli.go` — `CLI` struct implementing the subcommands (`import`, `export`, `search`, `list`, `delete`, …). Shares the `Database` implementation with the GUI.
- `model.go` — shared domain types (`Position`, `Board`, `Cube`, `Match`, `Game`, `Move`, `PositionAnalysis`, FSRS `AnkiCard`/`AnkiDeck`, `Tournament`, …) plus constants like `DatabaseVersion`, color/bar indices, decision-type enums.
- `epc.go` — Effective Pip Count engine; embeds `gnubg_os6.bd` (one-sided 6-point bearoff DB) via `//go:embed`.
- `config.go` — `Config` struct persisted as JSON at XDG config path `blunderDB/config.yaml` (window size, last DB path).

Match/position import parsers are **external modules** (own repos under `github.com/kevung/…`): `xgparser` (eXtreme Gammon `.xg`/`.xgp`), `gnubgparser` (GnuBG `.sgf`), `bgfparser` (BGBlitz `.bgf`). Jellyfish `.mat` parsing is handled in this repo. Position text files are a JSON-per-line format produced by the app itself.

Key backend invariants worth knowing before editing:
- Positions are stored with a canonical SHA-256 hash (see `canonical_hash_test.go`) to dedup across imports — always go through `SavePosition`, which handles the lookup.
- A single mutex (`Database.mu`) serializes writes; import cancellation uses an atomic flag (`importCancelled`).
- Schema changes require incrementing `DatabaseVersion` in `model.go` **and** adding a migration path (`CheckVersion` in `db_migration.go`, table/index DDL in `db_schema.go`). Cover the migration with a test in `migration_test.go`.

### Frontend (Svelte 5 + Vite, in `frontend/`)

- `frontend/src/App.svelte` — root component that mounts the toolbar, board, command line, and the modals/panels under `frontend/src/components/`. Kept thin — feature logic lives in the panels and stores.
- `frontend/src/stores/` — Svelte stores, one per feature area (positions, analysis, collection, tournament, Anki, EPC, search history, UI, gammon/takepoint tables, etc.). Cross-component state lives here, not in props.
- `frontend/src/commandProcessor.js` — parses the in-app command line (see `CommandLine.svelte`); a large source of user-facing behavior.
- `frontend/src/components/Board.svelte` — board rendering uses `two.js`.
- `frontend/wailsjs/` — **auto-generated** Go↔JS bindings. Do not hand-edit. They regenerate when `wails dev`/`wails build` sees changes to exported methods on bound structs (`App`, `Database`, `Config`). After adding a backend method, restart `wails dev` so the `.js`/`.d.ts` in `wailsjs/go/main/` are refreshed.

### Svelte 5 — store/effect rule

In this project, any store access inside a Svelte 5 component **must** use the auto-subscribe syntax `$storeName` or a `$effect(() => { const v = $storeName; ... })`. **Do not** use `.subscribe()` inside a component (rare exceptions must be justified in the commit message). Reason: `.subscribe()` callbacks capture stale closures and their internal dependencies (`$otherStore`, `get(x)`) are invisible to the Svelte compiler's dependency tracker — this caused the reactivity bugs documented in `tasks/ui-reactivity/` after the Svelte 5 migration.

### CLI/GUI parity

The CLI reuses the same `Database` methods the GUI calls over Wails. When adding DB functionality, prefer putting the logic on `Database` in `db.go` and exposing it from both `cli.go` (as a subcommand) and the frontend (it will auto-bind). Don't fork logic between a CLI-only helper and a GUI-only method.

## Notes & Gotchas

- Wails drag-drop on Linux: `DisableWebViewDrop` must stay `false` (bug #4743 — see comment in `main.go`).
- Linux WebKit GPU policy is forced to `Never` for stability.
- `*.db` files in the repo root (`a.db`, `c.db`, `testBG.db`, `Quiz*.db`) are user sample databases — do not commit changes to them and do not treat them as fixtures.
- Historical design/implementation notes have been moved to `doc/archive/` — consult them when touching the related subsystem but don't assume they reflect current code.
- **Document size rule.** Plans, task sheets, and design notes must stay ≤500 lines each. Split long documents into a README index + per-topic files.
- **v2.0.0 schema shape.** `DatabaseVersion = "2.0.0"`. Key ideas: Zobrist hash column + unique index on `position` for deduplication across imports; denormalized scalar filter columns on `position` (`decision_type`, `dice_1/2`, `pip_diff`, `off_1/2`, `back_checkers_1/2`, `no_contact`, `occupancy_1/2`, `point_mask_1/2`, …) and on `analysis` (`cube_error`, `best_move_equity_error`, `player1_win_rate`, …); bitboard occupancy/point-mask columns enable fast integer pre-filter for checker-structure pattern searches; WAL journal mode + tuned PRAGMAs (cache_size, synchronous=NORMAL, temp_store=MEMORY). See `DATABASE_OPTIMIZATION_PLAN.md` and `tasks/` for the full plan and per-phase task sheets.
