# Phase B (optional) — `blunderdb migrate` SQLite → PostgreSQL

**Goal.** A subcommand that copies a single-user SQLite database into a
PostgreSQL backend under a chosen `tenant_id`. Lets desktop users
"upload" their existing database into a server deployment.

**Estimate.** 2-3 days. **Risk.** Medium. **PRs.** 1.

**Prerequisites.** [P3](03-postgres-backend.md), [P4](04-session-scope.md).

## Why

- Desktop users have years of analysed positions in `.db` files. The
  server mode is useless to them if there's no migration path.
- Same logic could be reused later (Postgres → SQLite export, e.g. a
  user backing up their tenant) but that's a future stretch.

## Usage

```
blunderdb migrate \
    --from sqlite:///path/to/user.db \
    --to   postgres://user:pass@host:port/db?sslmode=disable \
    --tenant-id 12345 \
    [--batch-size 1000] \
    [--dry-run]
```

Behaviour:
- Opens both backends through `storage.Open(...)`.
- For each table in dependency order (positions → analyses → matches →
  games → moves → move_analyses → comments → collections →
  collection_position → tournaments → anki_deck → anki_card →
  filter_library → search_history → command_history → session metadata),
  streams rows from SQLite, rewrites IDs (auto-assigned by Postgres), and
  inserts under the given `tenant_id`.
- Reports progress as NDJSON to stdout.
- Wraps the whole migration in a single Postgres transaction — atomic
  rollback if anything fails.

## ID remapping

SQLite primary keys are local. Postgres assigns new IDs via `BIGSERIAL`.
The tool maintains an `oldID → newID` map per table (in memory; 38 k
positions × 16 bytes ≈ 600 KB → fine).

Foreign keys are rewritten on the fly using the maps.

## Dedup behaviour

Postgres has a `UNIQUE (tenant_id, zobrist_hash)` index on `position`. If
the destination tenant already has a position with the same Zobrist hash,
the migration:
- **Default**: errors out with a clear message; user must reset the tenant
  or rename.
- **`--on-conflict skip`**: re-uses the existing position ID, maps the
  source ID to it, and proceeds.
- **`--on-conflict overwrite`**: dangerous, rejected initially. Document
  but do not implement.

## Steps

- [ ] Add `internal/cli/cmd_migrate.go` (or
      `cmd_call.go` extension — but `migrate` is opinionated enough to
      warrant a dedicated subcommand).
- [ ] Implement table-by-table copy with progress reporting.
- [ ] Maintain ID remap; rewrite FKs as we go.
- [ ] Handle `scope` rewriting on session-like tables: source SQLite has
      `scope = ''`; destination Postgres needs `scope = "<tenant_id>"`
      (or empty if the user wants tenant-global state, but typically
      session state moves under the tenant scope).
- [ ] Handle the `metadata` table specially:
      - Globals like `database_version`, `met_table_id` → not copied
        (Postgres has its own).
      - Session-like (`session_*`, `last_visited_*`) → copy under the
        tenant scope.
- [ ] `--dry-run` simulates the migration and reports counts without
      writing.

## Tests

- [ ] `migrate_test.go`: take a fixture SQLite DB (`testdata/v2_8_0_small.db`
      with ~100 positions, 3 matches, 1 collection, 1 anki deck), migrate
      into a `testcontainers` Postgres under `tenant_id = 1`, verify row
      counts and a spot-check of a position's analysis hash.
- [ ] Conflict scenario: migrate same DB twice into same tenant →
      first succeeds, second fails with clear error (or skips with
      `--on-conflict skip`).
- [ ] Two tenants in same Postgres: migrate the same SQLite under
      `tenant_id = 1` and `tenant_id = 2`. Independent rows.

## Gotchas

1. **Large `analysis.data` blobs**. ~5 KB each, 38 k rows = ~190 MB
   transferred. Stream in batches; don't load all into RAM.
2. **Stable ordering**. When iterating SQLite tables, use `ORDER BY id`
   so the remap is deterministic.
3. **Wire compression**. The migration runs locally typically; no
   network compression needed. If running across a WAN, document
   `sslmode=require` and `compression=on` flags (Postgres 14+).
4. **Failure recovery**. If the migration aborts mid-way, the
   destination Postgres transaction rolls back fully. The user re-runs.
   No partial state to clean up.
5. **Migrating a v2.7.0 SQLite** (pre-scope) into a v2.8.0 Postgres: the
   tool migrates the SQLite to v2.8.0 first (`storage.Migrate`), then
   copies. Document the prerequisite or do it transparently.

## Documentation

- `CLI_USAGE.md`: dedicated section.
- `examples/migrate-sqlite-to-postgres.sh` end-to-end demo.

## Verification

- [ ] Fixture migration test green.
- [ ] Manual: migrate a ~200 MB user DB, total time < 60 s on local
      Postgres.
- [ ] Conflict cases handled with clear errors.

## PR layout

Single PR: `feat(cli): blunderdb migrate sqlite → postgres tool`.

## Done — implementation notes (core)

- `pkg/blunderdb/migrate`: backend-agnostic `Run(ctx, src, dst storage.Storage,
  scope string, opts) (Report, error)` reads every covered family from `src`
  and writes it to `dst` through the Storage interface, remapping
  position/match/game/tournament ids (and the FKs that reference them) via
  in-memory `oldID→newID` maps. All destination writes happen in one
  `dst.BeginTx` transaction — atomic; a failed run commits nothing.
- **Covered:** positions, analyses, comments, matches (games + moves),
  tournaments (+ match links), collections (+ membership).
- **Conflict guard:** with no `--on-conflict`, aborts if the destination scope
  already has positions; `--on-conflict skip` proceeds (positions dedup by
  Zobrist inside `PositionStore.Save`). `--dry-run` counts the source only.
- `pkg/blunderdb/migrate/cli.go` `RunCLI` parses
  `--from/--to/--tenant-id/--dry-run/--on-conflict/--batch-size` (batch-size
  reserved), opens SQLite + PostgreSQL, runs both `Migrate()` (upgrades a
  v2.7.0 source first), and streams NDJSON `{"event":…,"report":…}`. Dispatched
  from `main.go` like `serve`/`call`.
- Tests: `migrate_postgres_test.go` (`//go:build postgres`, testcontainers) —
  seed a SQLite DB (real XG match + a collection), migrate into PG under
  tenant "1", assert the report + iterated dest counts match the source, the
  conflict guard fires on re-run, and a second tenant is isolated; plus a
  default-tag dry-run check.

**Conflict-detection note.** The strict per-position "error on duplicate
Zobrist" of the original design is approximated by the empty-scope guard:
detecting per-row conflicts would need an `Exists` probe per position, and the
common case (fresh tenant) has none. `--on-conflict overwrite` remains
unimplemented by design.

**Deferred (needs P4 session-scope or low migration value):** anki decks/cards,
filter library, search/command history, session/metadata. The Postgres
`MetadataStore.Counts` is also still a stub (P3 PR6-8), so the tool and its test
avoid it.

## Future work (not in this phase)

- Postgres → SQLite (tenant export).
- Selective migration (only matches matching a filter).
- Parallel migration of multiple tenants from a directory of SQLite files.
- The deferred app-state families, once P4 lands.
