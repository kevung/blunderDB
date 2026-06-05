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

When you add a migration, also fold the change into `001_initial_v2_7_0.sql` (so
fresh databases get it directly), have the migration bump `database_version` in
`metadata`, bump `domain.DatabaseVersion` if schema-visible, and extend the
migration test.

## Multi-tenancy

Every domain table has `tenant_id BIGINT NOT NULL`. The application filters by
`tenant_id` on every query; this is mandatory regardless of whether the
optional Row-Level Security policies (see `../RLS.md`) are enabled.

The `metadata` table is database-level infrastructure (it holds the schema
version) and is **not** tenant-scoped.
