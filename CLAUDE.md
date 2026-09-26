# CLAUDE.md

Working rules and invariants for this repository — not an architecture tour. This file is
re-read at every turn of every agent: anything added here is paid everywhere, so a rule that
concerns one subsystem goes in a `CLAUDE.md` of that directory (`frontend/`, `doc/`,
`pkg/blunderdb/engine/gammonnet/`), loaded only when working there.

The architecture lives where it is documented: `ARCHITECTURE.md` (mode dispatch, layering, an
import's path), the package doc of `pkg/blunderdb/storage/storage.go` (persistence contract),
each package's `doc.go`, `CONTEXT.md` (domain glossary), `docs/adr/` (decisions),
`doc/source/mode_headless.rst` (server mode), `CLI_USAGE.md` (CLI reference).

## Project

blunderDB is a backgammon blunder database: a **Wails v2 desktop app** (Go + Svelte 5 / Vite)
whose single binary also runs headless, dispatched on `os.Args[1]` in `main.go`: no argument →
GUI; `serve` → HTTP daemon (SQLite or multi-tenant PostgreSQL); `call` → in-process dispatcher
over the same handlers; `migrate` → SQLite → PostgreSQL; anything in `handlers()` of
`internal/cli/cli.go` → CLI. That table is the only list of commands (`cli.IsCommand`,
`cli.CommandNames()`); never add a second one.

## Build, run, test

```bash
make dev                          # wails dev -tags webkit2_41 (webkit2gtk-4.1)
make build                        # → build/bin/blunderDB
go test ./...                     # Go; -run TestNameRegex for one test
cd frontend && npm test           # vitest
```

CI also enforces `go vet`, `go test -race`, `golangci-lint` (`.golangci.yml`), `govulncheck`,
`npm run lint`, `npm run format:check` (prettier fails the build), `npm run test:e2e`.
`cmd/serve/` builds the daemon alone (pure Go, no CGO, no Wails) for `Dockerfile.serve`.

Toolchain: Go and Node versions are stated once per workflow (`env: GO_VERSION` /
`NODE_VERSION`). `go.mod` states `go 1.26.0`, the minimum a dependency requires — gammonGo
embeds `pkg/blunderdb/server` and inherits it, so never raise it without a dependency that
demands it. Wails CLI and library v2.10.2.

Tests live beside the code; fixtures under `testdata/`. The `database` and `cli` test packages
`chdir` to the repo root in `TestMain`. Both storage backends pass the contract suite in
`pkg/blunderdb/storage/storagetest/`.

## Workflow

Every change is made in its own worktree, with an **absolute** path (a relative one can land
inside the repo), then merged back:

```bash
ROOT=$(git rev-parse --show-toplevel); WT="$ROOT/../blunderDB-<feature>"
git worktree add "$WT" -b feat/<feature>          # work + commit in $WT
cd "$ROOT" && git merge feat/<feature>
git worktree remove "$WT" && git branch -d feat/<feature>
```

A fresh worktree does not build: `main.go` embeds `frontend/dist`, which git ignores —
`mkdir -p "$WT/frontend/dist" && touch "$WT/frontend/dist/index.html"` for backend work.
Never `git reset --hard` on `main`: parallel sessions merge into it; undo with `git revert`.

**A user-visible feature ships with its documentation** in the same branch: the entries in
`doc/source/raccourcis.rst` / `manuel.rst` / `cmd_mode.rst` and their eight `.po` (rules in
`doc/CLAUDE.md`). A new CLI command also lands in `CLI_USAGE.md` (`go run ./cmd/cli-doc-gen`)
and `cli.rst`; `scripts/doc-inventory.sh` reports a gap — fix it by writing the text, never by
widening the script.

**Release**: the `release-blunderdb` skill (`scripts/release.sh <version>`). `DatabaseVersion`
in `pkg/blunderdb/domain/` is independent of the app version.

## Context budget (ADR-0055)

- A batch of issues goes through `/traiter-lot`: one fresh `ouvrier` sub-agent per issue or
  code zone; the orchestrator reads, decides, delegates and keeps verdicts only.
- Read a range, not a file: `grep -n`, then `sed -n a,b` or `Read` with `offset`/`limit`.
  A suite returns its failures, never its log. No `sleep`.
- Every `Agent` call names its `model`: Sonnet when a mistake shows up red, Opus when it would
  be silent (design, review, diagnosis, migrations, engine arithmetic).
- A comment says why, never the history: no issue number, date or narrative in code.
- `.claude/hooks/budget.py` holds these mechanically; `scripts/cout-tokens.py` measures them.

## Invariants

A violation is a bug even when every test passes.

- **Positions are identified by their Zobrist hash** (per tenant). Write through
  `SavePosition`; `SaveIndividualPosition` when the user brings a position on its own.
  Provenance and the session rules (`has_jacoby`, `has_beaver`) are not part of the hash, but
  their keys are still drawn in `engine.init`: changing that stream rehashes every database
  (ADR-0001, ADR-0028).
- **The retention predicate `positionIsHeldSQL` is written three times** — `database/db_match.go`
  (run by GUI and CLI), `storage/sqlite/matches_sqlite.go`, `storage/postgres/matches_postgres.go`
  — and stays identical up to SQL dialect. So does the orphan purge on match deletion.
- **A schema change** bumps `DatabaseVersion` and adds a `migrate_X_to_Y` step to
  `migrationSteps` (`db_migration.go`; `TestMigrationSteps_ContinuousChain` requires an unbroken
  chain), DDL in `db_schema.go`, the PostgreSQL side under `storage/postgres/migrations/`, and a
  test in `migration_test.go`. Then `scripts/build-demo-db.sh` (`TestDemoDatabaseIsCurrent`).
- **The serve daemon performs no authentication**: it trusts `X-Tenant-ID` behind an
  authenticating proxy. Never add auth to the engine, never weaken the warnings (ADR-0005).
- **Concurrency**: `Database.mu` is an RWMutex over the legacy wrapper (not reentrant); the
  storage backends have no global lock (pooled connections, one transaction per operation).
  Import cancellation is context-based (`beginCancellableImport`/`CancelImport`).
- **CLI/GUI/server parity**: logic goes on `Database` or the storage contract and is exposed to
  all three; never a mode-specific fork.
- **Nothing is recorded on the recipient's side**: a watermark is written by the producer at
  export; opening, reading or importing a database writes nothing (ADR-0007). Exports copy
  metadata through the `issuance.Carried` allow-list, never by exclusion.
- **One equity scale leaves the engine**: money points at money play, normalised equity (±1 =
  the current cube) at a match score. The engine's internal scales never reach storage or
  display (ADR-0019; the engine's own invariants: `engine/gammonnet/CLAUDE.md`).

## Gotchas

- Wails drag-drop on Linux: `DisableWebViewDrop` stays `false` (see `internal/gui/run.go`).
- `:memory:` is test-only; `sqlite.ConfigurePool` pins it to one connection. PRAGMAs live in
  `storage/sqlite/sqlite.go`.
- `internal/gui/demo.db.gz` is generated by `scripts/build-demo-db.sh`, with fictional names only.
- Go tools write temporaries to `/tmp`, a tmpfs that can fill up and break every shell command:
  set `GOTMPDIR` outside it for heavy builds.
- `tasks/` holds finished task sheets as history; open follow-ups live in `tasks/BACKLOG.md`.
- CI runs on three schedules (`build.yml`, `nightly.yml`, `fuzz.yml`); each header says why a
  check lives there — read it before adding a fourth.
- `.claude/settings.json` is shared (mattpocock skills plugin, context-budget hook);
  `.claude/settings.local.json` is personal and gitignored.
