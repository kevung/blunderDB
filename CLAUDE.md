# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

blunderDB is a backgammon blunder analysis tool. It is a **Wails v2 desktop application** (Go backend + Svelte 5 / Vite frontend) distributed as a single binary that runs in either GUI or CLI mode depending on the invocation.

## Build & Run

All commands run from the repo root unless stated.

```bash
# Run the GUI in dev mode (hot-reload frontend via Vite)
wails dev

# Production build → build/bin/blunderDB
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
./blunderDB import --db mymatches.db --type match --file game.xg
./blunderDB list --db mymatches.db --type stats
```

## Claude Code setup

`.claude/` is checked in so every machine working on this project gets the same
tooling:

- `.claude/settings.json` — shared config. Registers the `mattpocock`
  marketplace and enables the `mattpocock-skills` plugin (~22 engineering
  skills, invoked as `/mattpocock-skills:<name>`). Claude Code offers to
  install it once you trust the folder, so a fresh clone needs network access
  the first time.
- `.claude/skills/` — project skills committed directly (e.g.
  `release-blunderdb`). No settings needed; they are auto-discovered.
- `.claude/settings.local.json` — personal, per-machine overrides. Gitignored;
  keep machine-specific preferences (model, theme, extra permissions) there and
  out of `settings.json`.

## Development Workflow

For **every new modification**, work in an isolated git worktree to avoid conflicts — never edit directly on the shared checkout. The cycle is: create a worktree, do the work and commit there, merge the branch back, then remove the worktree to clean up.

Give `git worktree add` an **absolute** path: a `../blunderDB-<feature>` relative
to the wrong cwd silently creates the worktree *inside* the repo.

```bash
# 0. Anchor on the repo root so the paths below cannot drift with the cwd
ROOT=$(git rev-parse --show-toplevel)
WT="$ROOT/../blunderDB-<feature>"

# 1. Create a worktree on a fresh branch for the change
git worktree add "$WT" -b feat/<feature>

# 2. Work + commit inside that worktree
cd "$WT"
# … edit, test …
git add -A && git commit -m "feat: <feature>"

# 3. Merge the branch back into the base branch
cd "$ROOT"
git merge feat/<feature>

# 4. Clean up the worktree (and the branch once merged)
git worktree remove "$WT"
git branch -d feat/<feature>

# Sanity check: only the main checkout should remain
git worktree list
```

## Tests

```bash
go test ./...                     # all Go tests
go test ./tests/...               # the secondary test package
go test -run TestNameRegex ./...  # single test
```

Tests live in the package directories alongside the code they exercise — most are in `pkg/blunderdb/database/` (e.g. `migration_test.go`, `export_test.go`, `gnubg_import_test.go`), with others in `pkg/blunderdb/engine/`, `internal/cli/`, and `./tests/` (integration-style). Many tests rely on fixture files under `testdata/`, `gnubg/`, and on the embedded `gnubg_os6.bd` bearoff database; the `database` and `cli` test packages `chdir` to the repo root via a `TestMain` so those repo-root-relative paths resolve.

## Documentation

Sphinx docs live under `doc/` (French + English). Build with `cd doc && python build.py` (requires `doc/requirements.txt` and LaTeX for the PDF build). GitHub Pages publishes from `gh-pages` on tag pushes.

## Release Process

Use `scripts/release.sh <version>` — it updates the version string in three places (`doc/source/conf.py`, `frontend/src/stores/metaStore.js`, optionally `doc/source/index.rst` changelog) and creates a commit + tag. Pushing the tag triggers the CI matrix build and publishes binaries/PDFs as a GitHub release. The `DatabaseVersion` constant in `pkg/blunderdb/domain/` is independent — bump it only when the SQLite schema changes (migrations live in `pkg/blunderdb/database/db_migration.go`).

## Architecture

### Backend (Go)

The backend is split into importable packages under `pkg/blunderdb/` and
`internal/`, plus a thin `package main` at the repo root. (The split is the
result of the `tasks/headless/` P1 refactor; the `aliases.go` files noted
below are convenience re-exports kept intentionally so the `database`/`cli`
packages can use unqualified `domain` names — not a pending migration.)

- **Repo root (`package main`)** — `main.go` dispatches CLI vs GUI and holds
  the Wails `//go:embed` directives (`frontend/dist`, app icon); `config.go`
  is the `Config` struct (XDG-persisted JSON at `blunderDB/config.yaml`:
  window size, last DB path); `logging.go` configures `slog`. `main.go`
  stays at the repo root because Wails builds the `main` package from the
  project root and `//go:embed` patterns cannot use parent paths.
- **`pkg/blunderdb/domain/`** (`package domain`) — dependency-free domain
  types (`Position`, `Board`, `Cube`, `Match`, `Game`, `Move`,
  `PositionAnalysis`, FSRS `AnkiCard`/`AnkiDeck`, `Tournament`, …),
  constants (`DatabaseVersion`, color/bar indices, decision-type enums) and
  the pure `Position` predicate methods.
- **`pkg/blunderdb/engine/`** (`package engine`) — `bitboards.go`,
  `zobrist.go`, and `epc.go` (the Effective Pip Count engine, which embeds
  `gnubg_os6.bd` via `//go:embed`). Imports only `domain`.
- **`pkg/blunderdb/database/`** (`package database`) — the `Database` struct
  and the whole persistence layer:
  - `db.go` — connection lifecycle (`NewDatabase`, `SetupDatabase`,
    `OpenDatabase`, `applyPragmas`, `Conn`, `Close`). Uses
    `modernc.org/sqlite` (pure-Go SQLite, no CGO). The GUI opens the user's
    `.db` file directly; `:memory:` is only used by tests (which is why
    `ConfigurePool` pins it to a single connection — each pooled connection
    would otherwise be a separate empty database).
  - `db_schema.go` — table/index DDL. `db_migration.go` — `CheckVersion`
    and per-version upgrade paths.
  - `db_position.go`, `db_analysis.go`, `db_match.go`, `db_tournament.go`,
    `db_collection.go`, `db_comment.go`, `db_anki.go`, `db_session.go`,
    `db_met.go` — domain-specific persistence.
  - `db_search.go`, `db_filter_match.go`, `db_stats.go` — query, filter and
    aggregate logic.
  - `db_import_*.go` and `db_export.go` — import/export pipelines for XG,
    GnuBG, BGF, native `.db`, and JSON.
  - `aliases.go` re-exports `domain` names into the package.
- **`internal/gui/`** (`package gui`) — `app.go` (`App` struct, bound to the
  frontend: file/directory dialogs, drag-drop file reading, clipboard via
  xclip/wl-copy/osascript/PowerShell, alerts) and `run.go` (the Wails
  bootstrap that wires `Bind`).
- **`internal/cli/`** (`package cli`) — `cli.go` (`CLI` struct, `Run`
  dispatcher, shared helpers) and one `cli_<cmd>.go` per subcommand
  (`cli_import.go`, `cli_export.go`, `cli_list.go`, …); `aliases.go`
  re-exports `domain`/`database` names.

Match/position import parsers are **external modules** (own repos under `github.com/kevung/…`): `xgparser` (eXtreme Gammon `.xg`/`.xgp`), `gnubgparser` (GnuBG `.sgf`), `bgfparser` (BGBlitz `.bgf`). Jellyfish `.mat` parsing is handled in this repo. Position text files are a JSON-per-line format produced by the app itself.

Domain vocabulary and the decisions behind it live in `CONTEXT.md` (glossary) and `docs/adr/` — read those before changing how positions are identified, where they came from, or what survives a match deletion.

Key backend invariants worth knowing before editing:
- Positions are stored with a canonical hash (Zobrist; see `pkg/blunderdb/database/canonical_hash_test.go`) to dedup across imports — always go through `SavePosition`, which handles the lookup. Provenance (`Position.IndividuallyImported`) is deliberately **not** part of that hash: hashing it would split one position into two rows. Use `SaveIndividualPosition` when the user brings a position in on its own.
- Deleting a match purges the positions it referenced that nothing else holds. The retention predicate (`positionIsHeldSQL`) is stated in **three** places — `database/db_match.go` (the copy the GUI and CLI run), `storage/sqlite/matches_sqlite.go` and `storage/postgres/matches_postgres.go`. Keep them identical.
- A single mutex (`Database.mu`) serializes writes; import cancellation uses an atomic flag (`importCancelled`).
- Schema changes require incrementing `DatabaseVersion` in `pkg/blunderdb/domain/` **and** adding a migration path (`CheckVersion` in `db_migration.go`, table/index DDL in `db_schema.go`). Cover the migration with a test in `pkg/blunderdb/database/migration_test.go`.

### Frontend (Svelte 5 + Vite, in `frontend/`)

- `frontend/src/App.svelte` — root component that mounts the toolbar, board, command line, and the modals/panels under `frontend/src/components/`. Kept thin — feature logic lives in the panels and stores.
- `frontend/src/stores/` — Svelte stores, one per feature area (positions, analysis, collection, tournament, Anki, EPC, search history, UI, gammon/takepoint tables, etc.). Cross-component state lives here, not in props.
- `frontend/src/commandProcessor.js` — parses the in-app command line (see `CommandLine.svelte`); a large source of user-facing behavior.
- `frontend/src/components/Board.svelte` — board rendering uses `two.js`.
- `frontend/wailsjs/` — **auto-generated** Go↔JS bindings. Do not hand-edit. They regenerate when `wails dev`/`wails build` sees changes to exported methods on bound structs. The bound structs live in different packages, so the bindings are namespaced by package: `App` → `wailsjs/go/gui/`, `Database` → `wailsjs/go/database/`, `Config` → `wailsjs/go/main/`. After adding a backend method, restart `wails dev` so the `.js`/`.d.ts` are refreshed.

### Svelte 5 — store/effect rule

In this project, any store access inside a Svelte 5 component **must** use the auto-subscribe syntax `$storeName` or a `$effect(() => { const v = $storeName; ... })`. **Do not** use `.subscribe()` inside a component (rare exceptions must be justified in the commit message). Reason: `.subscribe()` callbacks capture stale closures and their internal dependencies (`$otherStore`, `get(x)`) are invisible to the Svelte compiler's dependency tracker — this caused the reactivity bugs documented in `tasks/ui-reactivity/` after the Svelte 5 migration.

### CLI/GUI parity

The CLI reuses the same `Database` methods the GUI calls over Wails. When adding DB functionality, prefer putting the logic on `Database` in `pkg/blunderdb/database/` and exposing it from both `internal/cli/` (as a subcommand) and the frontend (it will auto-bind). Don't fork logic between a CLI-only helper and a GUI-only method.

## Notes & Gotchas

- Wails drag-drop on Linux: `DisableWebViewDrop` must stay `false` (bug #4743 — see comment in `main.go`).
- Linux WebKit GPU policy is forced to `Never` for stability.
- `*.db` files in the repo root (`a.db`, `c.db`, `testBG.db`, `Quiz*.db`) are user sample databases — do not commit changes to them and do not treat them as fixtures.
- Historical design/implementation notes have been moved to `doc/archive/` — consult them when touching the related subsystem but don't assume they reflect current code.
- **Document size rule.** Plans, task sheets, and design notes must stay ≤500 lines each. Split long documents into a README index + per-topic files.
- **v2.0.0 schema shape.** `DatabaseVersion = "2.0.0"`. Key ideas: Zobrist hash column + unique index on `position` for deduplication across imports; denormalized scalar filter columns on `position` (`decision_type`, `dice_1/2`, `pip_diff`, `off_1/2`, `back_checkers_1/2`, `no_contact`, `occupancy_1/2`, `point_mask_1/2`, …) and on `analysis` (`cube_error`, `best_move_equity_error`, `player1_win_rate`, …); bitboard occupancy/point-mask columns enable fast integer pre-filter for checker-structure pattern searches; WAL journal mode + tuned PRAGMAs (cache_size, synchronous=NORMAL, temp_store=MEMORY). See `DATABASE_OPTIMIZATION_PLAN.md` and `tasks/` for the full plan and per-phase task sheets.
