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
- `create|import|export|identity|open|list|match|collection|anki|verify|vacuum|delete|help|version|info|edit|search|epc|analyze` → **CLI**. The
  names live in one place — `handlers()` in `internal/cli/cli.go`; `main.go`
  asks `cli.IsCommand`. Never re-introduce a second list there.

## Build & Run

All commands run from the repo root unless stated.

```bash
make dev      # wails dev  -tags webkit2_41  (hot-reload frontend via Vite)
make build    # wails build -tags webkit2_41 → build/bin/blunderDB
```

The `webkit2_41` tag matches webkit2gtk-4.1 (Arch, ubuntu-latest); plain
`wails build` targets webkit2gtk-4.0 (ubuntu-22.04 in CI). CI
(`.github/workflows/build.yml`) builds ubuntu-latest, ubuntu-22.04,
windows-latest, macos-latest (`darwin/universal`). Toolchain: Go 1.25.13 (CI and
`go.mod`), Node 23.4.0, Wails CLI v2.10.2 (library v2.10.1).

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

**A fresh worktree does not build at the repo root.** `main.go` embeds the compiled
frontend (`//go:embed all:frontend/dist`), and `frontend/dist/` is a build artefact that
git ignores — so a worktree that has never had a frontend build fails immediately, and
the error names the embed rather than the cause:

```
main.go:20:12: pattern all:frontend/dist: no matching files found
```

Either run a real build, or drop a placeholder in for backend work:

```bash
mkdir -p "$WT/frontend/dist" && touch "$WT/frontend/dist/index.html"
```

Nothing to clean up afterwards: the directory is gitignored.

**A user-visible feature ships with its documentation.** Any new command, shortcut,
panel, or filter must land in the same branch as its `doc/source/raccourcis.rst` /
`doc/source/manuel.rst` / `doc/source/cmd_mode.rst` entries **and** their eight
`.po` (see Documentation below — the online docs deploy from `main`). Undocumented
features have gone undiscovered for whole release cycles; don't add to that pile.

## Documentation

Sphinx docs live under `doc/` in **nine languages**: French is the source,
`doc/source/locale/` holds gettext translations (en, de, el, es, fi, it, ja, ru).
Build with `cd doc && python build.py` (requires `doc/requirements.txt` and LaTeX
for the PDF build). GitHub Pages publishes from `gh-pages` on tag pushes. Historical
design notes live in `doc/archive/` — consult when touching the related subsystem,
but they do not reflect current code.

**A modified `.rst` ships with its eight `.po` in the same commit.** The online
docs deploy from `main` and fall back to French for every untranslated string, so
a translation gap is a user-visible regression on eight sites, not release
polish. Refresh the catalogues, translate the empty `msgstr`, and check:

```bash
source .venv/bin/activate
scripts/doc-po-update.sh       # regenerates the eight catalogues, then repairs
scripts/doc-i18n-check.sh      # must end on "all translations complete"
```

Run `doc-po-update.sh` rather than sphinx-build/sphinx-intl by hand: it keeps
the gettext output path **relative** (an absolute one rewrites every `#:`
reference and buries the real diff), and it normalises the `python-format`
flag that `msgmerge` re-adds behind an existing `no-python-format` at every
update — gettext honours the last flag, so the false positive comes back each
time and makes `msgfmt` refuse the catalogue. Never re-wrap a `.po` with
`msgcat`: Babel and msgcat disagree on line breaks, so one pass rewrites the
whole file.

**Document size rule.** Plans, task sheets, and design notes stay ≤500 lines each.
Split long documents into a README index + per-topic files.

## Release Process

Use `scripts/release.sh <version>` — it updates the version in **four** places
(`doc/source/conf.py`, `frontend/src/stores/metaStore.js`, `wails.json`, optionally
the `doc/source/index.rst` changelog) and creates a commit + tag. Pushing the tag
triggers the CI matrix build and publishes binaries/PDFs as a GitHub release. Use
the `release-blunderdb` skill to drive the whole thing, including the doc audit.

The `DatabaseVersion` constant in `pkg/blunderdb/domain/` (currently **2.18.0**) is
independent of the app version — bump it only when the SQLite schema changes.

## Architecture in one screen

Backend packages, thinnest description that lets you find things:

- `main.go` (repo root, `package main`) — mode dispatch + Wails `//go:embed`
  (must stay at root: embed patterns can't use parent paths); `config.go`
  (XDG-persisted window/last-DB config); `logging.go` (slog).
- `pkg/blunderdb/domain/` — dependency-free domain types and constants
  (`Position`, `Match`, FSRS cards, `DatabaseVersion`).
- `pkg/blunderdb/engine/` — what is computed ABOUT a position: Zobrist hashing
  (the identity positions dedup on), bitboards, EPC (embeds `gnubg_os6.bd`), the
  match equity table (`met.go` — Kazaross-XG2 + Zadeh, `GnuBGGetME` the single
  entry point), and the two storage codecs (compact board, zstd analysis blob,
  and every derived scalar column). Read its package doc (`doc.go`) for the
  file-by-file map. Two subpackages, the two evaluators:
  `engine/race/` — bearoff race analysis: two-sided `.bd` reader (embeds
  `gnubg_ts0.bd`), win-probability estimation, money cube verdicts (ADR-0009);
  `engine/gammonnet/` — the neural evaluator, ~5 000 lines and the largest thing
  in the tree: a Go port of gammonNet's encoding, network (AVX2/pure-Go kernel),
  expectiminimax search and Janowski cube model (ADR-0011, ADR-0024). Its
  arithmetic is a contract: read its package doc and `cube.go`'s header first.
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
- `pkg/blunderdb/issuance/` — marking a database with its origin (signed
  watermark + issuer identity) and wrapping an export in an encrypted container.
  **Pure**: no SQL, no schema; its glue is `database/db_issuance.go` and it
  stores one canonical signed JSON document in a `metadata` row. Read its
  package doc and ADR-0007 before touching it.
- `internal/gui/` — Wails `App` (dialogs, clipboard, drag-drop) + bootstrap;
  `internal/cli/` — one `cli_<cmd>.go` per subcommand; `internal/server/` — the
  HTTP daemon (`routes.go`, `handlers_*.go`, middleware, metrics, `call.go`).
- `cmd/` — `serve` (headless entrypoint) plus dev-time tools that never ship in
  the binary: `blunderdb-loadtest`, `extract_gnubg_stats`, `calibrace` (fits
  `engine/race`'s correction against a TS-06-11 oracle), `train-analysis-dict`
  (regenerates the embedded zstd dictionary, ADR-0030).

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
  — see ADR-0001 and `CONTEXT.md`. Neither are the session's optional rules
  (`has_jacoby`, `has_beaver`): only an XGID ever sets them, so hashing them
  split one money position across two rows — ADR-0028. Both keys are still
  DRAWN in `engine.init` so every key after them keeps its value; a change to
  that stream rehashes every database ever written.
- **The retention predicate (`positionIsHeldSQL`) is stated in three places** —
  `database/db_match.go` (the copy the GUI and CLI run),
  `storage/sqlite/matches_sqlite.go`, `storage/postgres/matches_postgres.go`.
  Keep the *predicate* identical in all three (placeholders and boolean syntax
  differ by SQL dialect).
- **Schema changes** require bumping `DatabaseVersion` in `pkg/blunderdb/domain/`
  **and** a migration path (`CheckVersion` in `db_schema.go` — it only compares the major version — and
  a `migrate_X_to_Y` step registered in `migrationSteps` in `db_migration.go` —
  the registry `runMigrationChain` walks, which `TestMigrationSteps_ContinuousChain`
  requires to run unbroken from 1.0.0 to `DatabaseVersion` — DDL in
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
- **Nothing is recorded on the recipient's side**: a watermark is written by the
  producer, at export, on a file they are making. blunderDB must never write a
  registry, log, counter or lineage into a database because someone opened,
  read or imported it. Reviving any of that reopens a decision taken and
  reversed once already — see ADR-0007 before proposing it.
- **Exports carry an allow-list**: `ExportDatabase` copies metadata through
  `issuance.Carried`, never by exclusion — a document added to `metadata` next
  year must not travel to someone else's machine by default.
- **One equity scale leaves the engine**: money points at money play,
  **normalised equity** (gnubg's `mwc2eq`, ±1 = winning/losing the current
  cube) at a match score — the scale imported XG/GNUbg analyses already sit
  in, in the same columns, and the one the statistics read as EMG. gammonNet's
  own internals are two OTHER scales (`Decide` returns MWC, `Value` returns
  2×MWC−1); they stay inside the package, and `EquityScale`
  (`gammonnet/referential.go`) converts once at the domain edge. Never print
  or store either internal scale — at 5-away/5-away they are six times too
  small and look perfectly plausible. See ADR-0019.
- **The live cube curve has no plateau**: `janowskiEquity`/`levelLive` in
  `engine/gammonnet/cube.go` run from `(0, −L)` to `(1, +W)`, bending only at
  the breakpoints the cube state imposes. Clamping either tail to the cash
  equivalent (`max(dead, 1)`, `min(dead, −1)`) prices the retained cube at
  zero and makes `TooGood` unreachable unless the *cubeless* equity already
  exceeds a point — the panel then says "double, passe" on every real
  too-good position. That is a Jacoby rule, and Jacoby is already applied
  where it belongs, in `Decide`'s no-double payoffs. See ADR-0022; and the
  file is a port, so any change here lands in gammonNet's `gn_cube.c` and its
  spec §2 first, then in `testdata/cube_gold.bin`.
- **The cube efficiency is a branch coefficient, and the search reads the root's on
  purpose**: `DefaultEfficiency` returns one value per cube state (owned 0.566, centred
  0.688, opponent 0.687), each fitted against a different column of gammonNet's exact
  two-sided table — a deliberate divergence from gnubg and XG, which index it by
  position class. `SearchConfig.CubeX` is fixed at the root while `CubeOwner` is
  mirrored, and `Decide` prices `eDT` at the current owner's coefficient: both match
  `gn_search.c`/`gn_cube.c` line for line, so "fixing" either here manufactures a port
  divergence and turns the cube gold red. Measured (669 real decisions): 0.005
  normalised equity per leaf, no verdict flipped, no move changed. The correction is
  gammonNet's to write. See ADR-0029.
- **A shared optimisation is decided upstream, in gammonNet** (its ADR-0003). The criterion
  is measurable, not a matter of taste: *an optimisation is conceptual if its gain survives a
  change of language*. Conceptual ones — the shape of the algorithm — are written in
  gammonNet first, with their measurement, and this port follows. Implementation ones stay
  here without remorse: the AVX2 kernel is ours and has no business upstream. Measured on
  2026-09-02, five of this port's six non-network wins turned out to be language artefacts
  worth 0.007 % to 0.5 % in C — which is why the criterion is measured and not guessed.
- **The network kernel never fuses and never reassociates**: the batched evaluator
  vectorises over POSITIONS, one per SIMD lane, each lane accumulating over `j` in
  ascending order in float32, with multiply and add kept as two operations. No FMA
  (`vfmadd*`, `fmla`), no tree reduction, no float64 accumulation — the explicit
  `float32(a*b)` is a fusion barrier guaranteed by the Go spec, and it is on arm64 that
  it protects, since Go contracts there and never on amd64. Any new arithmetic path
  (a NEON kernel, a wider tile) must pass `kernel_identity_test.go` against the pure-Go
  fallback, `==` on every bit — the gold suites tolerate 1e-6 and would let an FMA
  through. A requested-but-unavailable kernel is an error at load, never a silent
  fallback. See ADR-0024.
- **Parallelism is production behaviour, not a test tool**: the analysis batch runs
  positions across `NumCPU` goroutines, each reusing one serial `Searcher`
  (`NewBatchSearcher`/`EvaluatePositionWith`); the live panel runs ONE search with
  `WithWorkers`. The two never stack — a batch path that also took the pool would ask for
  `NumCPU²` goroutines. Both stay bit-identical because the weighted sum over the 21 rolls
  is taken serially in ascending roll order: parallelism decides who computes each term,
  never the order they are added in.
- **One type scale**: components use the tokens in `frontend/src/style.css`
  (`--font-size-base/-small/-title`), never an absolute `font-size`, and form
  controls carry `font: inherit` — an input inherits neither size nor family, so
  setting only a size leaves it in the browser's control font. Hierarchy comes
  from weight and colour. Exceptions (chrome, statistics figures) are named in
  ADR-0008; the migration of existing components is gradual, the rule is not.
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
- `internal/gui/demo.db.gz` is generated, never hand-built: run
  `scripts/build-demo-db.sh` after every `DatabaseVersion` bump
  (`TestDemoDatabaseIsCurrent` fails otherwise). Fictional names only — the
  fixtures name real people and `scripts/demodb` disguises them (#162).
- `tasks/` holds finished task sheets (v2.0.0 optimization, headless refactor,
  stats parity…) kept as execution history; `tasks/FOLLOWUPS.md` lists still-open
  follow-ups.
