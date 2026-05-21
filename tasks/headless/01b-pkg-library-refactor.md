# P1 — Refactor into importable `pkg/blunderdb/`

**Goal.** Move all non-GUI Go code out of `package main` at the repo root
into a layered set of sub-packages under `pkg/blunderdb/`, so the engine is
usable as a library by external Go programs. The GUI binary (`wails build`)
and the CLI (`./blunderdb …`) keep working identically.

**Estimate.** 5-7 days. **Risk.** High (mass file move). **PRs.** 3.

**Prerequisites.** [P0](00-module-rename.md), [Phase A](01a-cli-extract.md).

## Target structure

```
pkg/blunderdb/
  domain/          ← types from model.go: Position, Board, Cube, Match,
                     Game, Move, Analysis, AnkiCard, AnkiDeck, Tournament,
                     Collection, …
  engine/          ← bitboards.go, zobrist.go, epc.go (with embed
                     gnubg_os6.bd), canonical-hash logic
  database/        ← Database type (wrapper kept for Wails Bind),
                     constructor, options, applyPragmas
  storage/         ← (empty in this phase, populated in P2)
  importers/       ← db_import_xg.go, db_import_gnubg.go, db_import_bgf.go,
                     db_import_db.go, db_import_common.go, db_export.go
  migration/       ← db_migration.go + DatabaseVersion constants
  session/         ← db_session.go
  search/          ← db_search.go + db_filter_match.go
  stats/           ← db_stats.go
  schema/          ← db_schema.go (DDL constants/builders)
  positions/       ← db_position.go
  analyses/        ← db_analysis.go
  matches/         ← db_match.go
  collections/     ← db_collection.go
  tournaments/     ← db_tournament.go
  comments/        ← db_comment.go
  anki/            ← db_anki.go
  met/             ← db_met.go (match-equity tables)

cmd/blunderdb/      ← main.go (GUI/CLI/serve dispatch)
internal/
  gui/             ← app.go (Wails-only)
  cli/             ← cli.go + cmd_<subcommand>.go (moved from repo root)
```

The repo root contains only `go.mod`, `go.sum`, `wails.json`, build
artefacts, fixture DBs, docs, scripts, and frontend files. **No Go source
file remains at the repo root**.

## Why a wrapper `Database` survives

The Wails frontend binds against the public method set of the `Database`
struct (`wailsjs/go/main/Database.*` is auto-generated). Rewriting the
frontend to consume `Storage` directly is out of scope here. Therefore
`pkg/blunderdb/database/Database` is preserved as a thin façade that holds
a `*sql.DB` (for now — until [P2](02-storage-interface.md) plugs in
`Storage`).

## Splitting into 3 PRs

**PR 1 — `pkg/blunderdb/{domain,engine}` move.**
- Domain types from `model.go` go to `pkg/blunderdb/domain/`.
- Engine code (`bitboards.go`, `zobrist.go`, `epc.go`) goes to
  `pkg/blunderdb/engine/`. The `//go:embed gnubg_os6.bd` directive follows
  the file — confirm the `.bd` file is moved to
  `pkg/blunderdb/engine/gnubg_os6.bd`.
- Update all callers (everything currently in `package main` at the root)
  to import the new packages.
- Tests of these modules move with them
  (`canonical_hash_test.go`, `bitboards_test.go`, `zobrist_test.go`,
  `epc_test.go`).

**PR 2 — `pkg/blunderdb/{database,positions,analyses,matches,collections,tournaments,comments,anki,met,session,search,stats,schema,migration}` move.**
- `db.go` core (`NewDatabase`, `SetupDatabase`, `OpenDatabase`,
  `applyPragmas`) → `pkg/blunderdb/database/`. Keep the type name
  `Database` and re-export it from the repo root via a type alias **only if
  the Wails binding needs it** (Wails reads the bound concrete type — check
  whether a type alias from `package main` to
  `database.Database` is enough; if not, `cmd/blunderdb/main.go` can do
  `type Database = database.Database` and bind that).
- Domain persistence files (`db_position.go`, `db_analysis.go`, etc.) move
  to their respective sub-packages. They keep operating on
  `*Database` (the wrapper).
- Migration code (`db_migration.go`) moves to `pkg/blunderdb/migration/`.
- Schema DDL (`db_schema.go`) moves to `pkg/blunderdb/schema/`.
- All tests at the repo root that depend on these (e.g.
  `migration_test.go`, `export_test.go`, `gnubg_import_test.go`,
  `canonical_hash_test.go`, etc.) move with their files.

**PR 3 — `cmd/blunderdb/` + `internal/{gui,cli}/` move.**
- `main.go` → `cmd/blunderdb/main.go`. Adjust the `wails.json` build hooks
  if they reference the root file path.
- `app.go` → `internal/gui/app.go`. Make sure the Wails binding still
  works (`wails dev` regenerates the JS shim).
- `cli.go` and `cli_*.go` (from Phase A) → `internal/cli/`. Rename package
  from `main` to `cli`. Re-import in `cmd/blunderdb/main.go`.
- `config.go` and `logging.go`: deferred decision — they are CLI-coupled.
  Keep them in `cmd/blunderdb/` for now; revisit in P6 when the server
  needs its own config.

## Gotchas

1. **`//go:embed build/appicon.png`** in `main.go` — the `build/` directory
   becomes relative to `cmd/blunderdb/`. Adjust the path or move the embed
   target.
2. **`//go:embed gnubg_os6.bd`** must be in the same directory as the file
   it's embedded into. Move both together.
3. **Test fixtures** (`testdata/`, `gnubg/`) — the tests that consume them
   pass relative paths. Either keep `testdata/` at the repo root and adjust
   the relative paths in the moved tests, or duplicate (do **not**
   duplicate). Recommended: keep `testdata/` at the repo root and have tests
   reference it via `filepath.Join("..", "..", "testdata", …)` (or use
   `runtime.Caller` to locate the repo root).
4. **`migrationProgress func(phase string, done, total int)`** field on
   `Database` is currently mutated by the GUI. After the move, expose it as
   a constructor option: `database.New(..., database.WithProgress(fn))`.
   Remove the public field.
5. **`Database` wrapper API kept stable.** No public method of `Database`
   may be renamed or have its signature changed in this phase. Wails
   bindings depend on names verbatim.
6. **`cliCommands` constant** (`main.go:25`) moves to `internal/cli/` and
   is consumed by the dispatcher in `cmd/blunderdb/main.go`.
7. **Module imports** — every Go file in the new layout uses
   `github.com/kevung/blunderdb/pkg/blunderdb/<sub>` imports. Goimports can
   help; run `goimports -w` on the whole tree after each PR.
8. **Sample DBs at the repo root** (`a.db`, `c.db`, `testBG.db`,
   `Quiz*.db`) stay where they are; they are user sample data.

## Verification (per PR)

After each PR:
- [ ] `go vet ./...` clean.
- [ ] `go test ./...` green.
- [ ] `go test ./tests/...` green.
- [ ] `wails build` produces a working binary; smoke-test:
  ```bash
  ./blunderdb help
  ./blunderdb create --db /tmp/x.db
  ./blunderdb import --db /tmp/x.db --type match --file testdata/<sample>.xg
  ./blunderdb info --db /tmp/x.db
  ```
- [ ] `wails dev` boots and the GUI loads a `.db` file end-to-end.

## Verification (overall)

After PR 3:

- [ ] A throwaway Go program outside this repo can do:
  ```go
  package main
  import (
      "context"
      "fmt"
      "github.com/kevung/blunderdb/pkg/blunderdb/database"
  )
  func main() {
      db, _ := database.New(context.Background(), database.MemoryDSN)
      defer db.Close()
      fmt.Println(db.GetDatabaseVersion())
  }
  ```
  And it builds + runs.

- [ ] `pkg/blunderdb/database/example_test.go` is added, demonstrating the
  public API via an `Example…` test that doubles as godoc documentation.

- [ ] No Go file remains at the repo root.

## Risks

- Tests at the repo root reference unqualified symbols (`Database`,
  `Position`, …). All files move together — but if mid-PR you leave some
  `_test.go` files behind, builds break. Mitigation: move tests in the
  same commit as the production code.
- Wails binding regeneration: confirm `wailsjs/go/main/*.{js,d.ts}` still
  matches the methods on `Database` after the move. If `Database` lives in
  another package, Wails may regenerate under `wailsjs/go/database/…`.
  Decision needed: either (a) keep `Database` as a type-aliased re-export
  in `package main` of `cmd/blunderdb`, or (b) update the frontend imports
  to the new generated path. (a) is less invasive.

## PR layout

1. `refactor(pkg): extract domain types and engine into pkg/blunderdb/`
2. `refactor(pkg): move database/persistence into pkg/blunderdb/`
3. `refactor(cmd): move binary entrypoint, GUI, CLI into cmd/ and internal/`
