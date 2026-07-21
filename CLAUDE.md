# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

It deliberately holds **working rules and invariants, not an architecture tour**. The
architecture is documented where it lives: the package doc of
`pkg/blunderdb/storage/storage.go` (persistence contract and backends), `CONTEXT.md`
(domain glossary), `docs/adr/` (decisions), `doc/source/mode_headless.rst` (server
mode, user-facing), and `CLI_USAGE.md` (CLI reference). Read those before changing
the subsystem they describe.

## Project Overview

blunderDB is a backgammon blunder analysis tool: a **Wails v2 desktop application**
(Go backend + Svelte 5 / Vite frontend) whose single binary also runs headless. One
executable, five modes, dispatched on `os.Args[1]` in `main.go`:

- No args → **GUI** (Wails desktop app)
- `serve` → **HTTP + JSON daemon** (SQLite or multi-tenant PostgreSQL backend)
- `call` → generic in-process dispatcher over the same handlers (scripting/tests)
- `migrate` → copy a SQLite database into PostgreSQL under a tenant
- `create|import|export|list|match|verify|delete|help|version|info|edit|search` → **CLI**

## Build & Run

All commands run from the repo root unless stated.

```bash
make dev      # wails dev  -tags webkit2_41  (hot-reload frontend via Vite)
make build    # wails build -tags webkit2_41 → build/bin/blunderDB
```

The `webkit2_41` tag matches webkit2gtk-4.1 (Arch, ubuntu-latest); plain
`wails build` targets webkit2gtk-4.0 (ubuntu-22.04 in CI). CI
(`.github/workflows/build.yml`) builds ubuntu-latest, ubuntu-22.04,
windows-latest, macos-latest (`darwin/universal`). Toolchain: Go 1.25.12 in CI
(`go.mod` says 1.25.10), Node 23.4.0, Wails CLI v2.10.2 (library v2.10.1).

`cmd/serve/` builds the daemon alone — pure Go, CGO disabled, no Wails — for the
container image (`Dockerfile.serve`).

## Tests

```bash
go test ./...                     # all Go tests
go test -run TestNameRegex ./...  # single test
cd frontend && npm test           # vitest
```

CI enforces more than the bare suites — run these before pushing anything nontrivial:
`go vet ./...`, `go test -race`, `golangci-lint` (v2.11.4, config `.golangci.yml`),
`govulncheck`, and on the frontend `npm run lint`, `npm run format:check` (prettier
**fails the build**), `npm run test:e2e` (Playwright).

Most Go tests live beside the code (`pkg/blunderdb/database/`,
`pkg/blunderdb/storage/…`, `internal/cli/`, `./tests/`). Fixtures live under
`testdata/`; the EPC engine embeds `pkg/blunderdb/engine/gnubg_os6.bd`. The
`database` and `cli` test packages `chdir` to the repo root via `TestMain` so
repo-root-relative fixture paths resolve. Both storage backends must pass the shared
contract suite in `pkg/blunderdb/storage/storagetest/`.

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

**A user-visible feature ships with its documentation.** Any new command, shortcut,
panel, or filter must land in the same branch as its `doc/source/raccourcis.rst` /
`doc/source/manuel.rst` / `doc/source/cmd_mode.rst` entries (French source only —
the 8 translations are refreshed at release time). Undocumented features have gone
undiscovered for whole release cycles; don't add to that pile.

## Documentation

Sphinx docs live under `doc/` in **nine languages**: French is the source,
`doc/source/locale/` holds gettext translations (en, de, el, es, fi, it, ja, ru).
Build with `cd doc && python build.py` (requires `doc/requirements.txt` and LaTeX
for the PDF build). GitHub Pages publishes from `gh-pages` on tag pushes. Historical
design notes live in `doc/archive/` — consult when touching the related subsystem,
but they do not reflect current code.

**Document size rule.** Plans, task sheets, and design notes stay ≤500 lines each.
Split long documents into a README index + per-topic files.

## Release Process

Use `scripts/release.sh <version>` — it updates the version in **four** places
(`doc/source/conf.py`, `frontend/src/stores/metaStore.js`, `wails.json`, optionally
the `doc/source/index.rst` changelog) and creates a commit + tag. Pushing the tag
triggers the CI matrix build and publishes binaries/PDFs as a GitHub release. Use
the `release-blunderdb` skill to drive the whole thing, including the doc audit.

The `DatabaseVersion` constant in `pkg/blunderdb/domain/` (currently **2.13.0**) is
independent of the app version — bump it only when the SQLite schema changes.

## Architecture in one screen

Backend packages, thinnest description that lets you find things:

- `main.go` (repo root, `package main`) — mode dispatch + Wails `//go:embed`
  (must stay at root: embed patterns can't use parent paths); `config.go`
  (XDG-persisted window/last-DB config); `logging.go` (slog).
- `pkg/blunderdb/domain/` — dependency-free domain types and constants
  (`Position`, `Match`, FSRS cards, `DatabaseVersion`).
- `pkg/blunderdb/engine/` — bitboards, Zobrist hashing, EPC (embeds `gnubg_os6.bd`).
- `pkg/blunderdb/storage/` — the persistence **contract**; backends
  `storage/sqlite/` (desktop/CLI) and `storage/postgres/` (serve daemon, RLS,
  tenant purge); shared contract tests in `storage/storagetest/`. Read this
  package's doc comment first.
- `pkg/blunderdb/database/` — the legacy SQLite-only `Database` wrapper the GUI
  and CLI run; delegates to `storage/sqlite`. Schema DDL in `db_schema.go`,
  migrations in `db_migration.go`, per-domain `db_*.go` files.
- `pkg/blunderdb/ingest/` — backend-agnostic import/export used by the daemon;
  `pkg/blunderdb/parser/` — position-text parsing shared by GUI/CLI/server;
  `pkg/blunderdb/migrate/` — SQLite→PostgreSQL copy; `pkg/blunderdb/server/` —
  `Bootstrap()` for in-process embedding by a trusted parent (gammonGo).
- `internal/gui/` — Wails `App` (dialogs, clipboard, drag-drop) + bootstrap;
  `internal/cli/` — one `cli_<cmd>.go` per subcommand; `internal/server/` — the
  HTTP daemon (`routes.go`, `handlers_*.go`, middleware, metrics, `call.go`).
- `cmd/` — `serve` (headless entrypoint), `blunderdb-loadtest`, `extract_gnubg_stats`.

Match/position parsers for external formats are separate modules
(`github.com/kevung/xgparser`, `gnubgparser`, `bgfparser`); Jellyfish `.mat`
handling lives in this repo (`database/db_mat_export.go`, `ingest/mat_export.go`).

Frontend: `frontend/src/App.svelte` stays thin; feature logic lives in
`components/` panels and one store per feature area under `stores/`.
`commandProcessor.js` parses the in-app command line;
`commandVocabulary.js` powers autocomplete and is locked to the processor by
`commandVocabulary.sync.test.js`. `Board.svelte` renders via two.js.

## Invariants

Violating one of these is a bug even if all tests pass:

- **Positions are identified by Zobrist hash** (per tenant) to dedup across
  imports. Always write through `SavePosition`; use `SaveIndividualPosition` when
  the user brings a position in on its own. Provenance
  (`individually_imported`) is sticky and deliberately **not** part of the hash
  — see ADR-0001 and `CONTEXT.md`.
- **The retention predicate (`positionIsHeldSQL`) is stated in three places** —
  `database/db_match.go` (the copy the GUI and CLI run),
  `storage/sqlite/matches_sqlite.go`, `storage/postgres/matches_postgres.go`.
  Keep the *predicate* identical in all three (placeholders and boolean syntax
  differ by SQL dialect).
- **Schema changes** require bumping `DatabaseVersion` in `pkg/blunderdb/domain/`
  **and** a migration path (`CheckVersion` in `db_migration.go`, DDL in
  `db_schema.go`, PostgreSQL side under `storage/postgres/migrations/`), covered by
  a test in `migration_test.go`.
- **The serve daemon performs NO authentication** — it trusts `X-Tenant-ID` and
  must run behind an authenticating reverse proxy. Never "fix" this by adding
  auth to the engine, and never weaken the warnings. See ADR-0005.
- **Concurrency**: `Database.mu` is an RWMutex over the legacy wrapper; the
  Storage backends have **no** global lock — they rely on pooled connections and
  per-operation transactions. Import cancellation is context-based
  (`beginCancellableImport`/`CancelImport` in `database/db.go`), not a flag.
- **CLI/GUI/server parity**: put DB logic on `Database` (or the Storage contract)
  and expose it to the frontend (auto-bound), the CLI, and the server handlers.
  Don't fork logic into a mode-specific helper.
- **Svelte 5 store rule**: inside components, always `$store` or
  `$effect(() => { const v = $store; … })` — **never** `.subscribe()` (stale
  closures, invisible to the compiler's dependency tracking; caused the
  post-migration reactivity bugs). Rare exceptions must be justified in the
  commit message.
- `frontend/wailsjs/` is **generated** (namespaced `gui`/`database`/`main`); never
  hand-edit; restart `wails dev` after changing exported bound methods.

## Notes & Gotchas

- Wails drag-drop on Linux: `DisableWebViewDrop` must stay `false` (bug #4743 —
  see comment in `internal/gui/run.go`); WebKit GPU policy is forced to `Never`.
- The GUI opens the user's `.db` file directly; `:memory:` is test-only, which is
  why `sqlite.ConfigurePool` pins it to a single connection (each pooled
  connection would otherwise be a separate empty database). PRAGMAs (WAL,
  `synchronous=NORMAL`, …) live in `storage/sqlite/sqlite.go`.
- `tasks/` holds finished task sheets (v2.0.0 optimization, headless refactor,
  stats parity…) kept as execution history; `tasks/FOLLOWUPS.md` lists still-open
  follow-ups.
