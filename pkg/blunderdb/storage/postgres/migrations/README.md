# PostgreSQL migrations

The PostgreSQL backend tracks its **own** forward migration chain, independent
of the SQLite one.

## Why no historical port

The SQLite backend carries 15 historical migrations (`db_migration.go`) that
upgrade pre-2.x databases written by older blunderDB releases. None of those
old databases exist as PostgreSQL databases — PostgreSQL is a new backend
introduced for the `serve` mode. Porting the historical chain would be dead
code.

So PostgreSQL **starts fresh at the terminal SQLite schema, v2.7.0**:

- `001_initial_v2_7_0.sql` — the complete v2.7.0 schema, multi-tenant.

## Forward chain

New schema changes are added as `NNN_description.sql` files, applied in
numeric order by `migrateForward` (see `migrate_postgres.go`). On every `Open`
/ `Migrate`, each file beyond the `001` baseline that is not yet recorded in the
`schema_migrations` table is applied (simple-protocol batch) and recorded. Make
every migration **idempotent** (`ADD COLUMN IF NOT EXISTS`,
`CREATE INDEX IF NOT EXISTS`, set-based backfills) so that applying it to a
freshly bootstrapped database — whose `001` baseline already contains the change
— is a harmless no-op.

- `002_is_cube_response.sql` — `position.is_cube_response` column + index, with a
  take/pass backfill from `move.cube_action` (mirrors
  `engine.IsResponseCubeAction`).
- `006_comment_position_index.sql` — `comment(tenant_id, position_id)` index for
  the comment-presence search filter (`co` / `xco`).
- `007_flagged.sql` — `position.flagged` column + index, the source-tool study
  mark (docs/adr/0006). No backfill is possible: the flag exists only in the
  source files, never in an already-imported database.
- `008_win_gammon_covering_index.sql` — extends the win/gammon combo search
  index with a trailing `position_id` column so the query's `p.id IN (SELECT
  position_id FROM analysis WHERE …)` subquery is answered from the index
  alone (fiche-05 T3). Index-only, like `006`.
- `009_luck_mp.sql` — `move.luck_mp` column, the luck of a roll in signed
  millipoints (docs/adr/0010). NULLable with no default: NULL means unknown,
  which is not the same as a neutral roll. No backfill is possible — luck
  exists only in the source files, never in an already-imported database.

When you add a migration, also fold the change into `001_initial_v2_7_0.sql` (so
fresh databases get it directly), have the migration bump `database_version` in
`metadata`, bump `domain.DatabaseVersion` if schema-visible, and extend the
migration test.

An **index-only** migration is the exception: it changes no column, no table and
no data, so nothing is schema-visible. It bumps neither `database_version` nor
`domain.DatabaseVersion` — recording a version with no counterpart in the Go
constant would desynchronise the two — and needs no migration-test entry.
`006` is one such migration.

## Multi-tenancy

Every domain table has `tenant_id BIGINT NOT NULL`. The application filters by
`tenant_id` on every query; this is mandatory regardless of whether the
optional Row-Level Security policies (see `../RLS.md`) are enabled.

The `metadata` table is database-level infrastructure (it holds the schema
version) and is **not** tenant-scoped.
