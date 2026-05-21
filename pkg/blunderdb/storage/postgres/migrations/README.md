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
numeric order. `002_*.sql` is reserved for P4 (the `scope` column on
`session` / per-session tables).

When you add a migration, also bump `domain.DatabaseVersion` if the change is
schema-visible, and extend the migration test.

## Multi-tenancy

Every domain table has `tenant_id BIGINT NOT NULL`. The application filters by
`tenant_id` on every query; this is mandatory regardless of whether the
optional Row-Level Security policies (see `../RLS.md`) are enabled.

The `metadata` table is database-level infrastructure (it holds the schema
version) and is **not** tenant-scoped.
