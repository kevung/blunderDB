# Row-Level Security & PostgreSQL tuning

## RLS — optional defence-in-depth

Multi-tenant isolation in blunderDB is **enforced by the application**: every
query filters by `tenant_id`. This is mandatory and sufficient.

Row-Level Security is offered as an optional second layer. It is **not** part
of `001_initial_v2_7_0.sql` and is **disabled by default**. It is implemented
in `rls_postgres.go`:

- **Install** — `Storage.ApplyRLS(ctx)` enables and **FORCE**s RLS on every
  tenant-scoped table and creates a fail-closed `tenant_isolation` policy:

  ```sql
  ALTER TABLE position ENABLE ROW LEVEL SECURITY;
  ALTER TABLE position FORCE  ROW LEVEL SECURITY;
  CREATE POLICY tenant_isolation ON position
      USING      (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint)
      WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint);
  ```

  `NULLIF(current_setting(..., true), '')` maps an unset/reset GUC (custom GUCs
  reset to `''`, not `NULL`) to `NULL`, so a connection without a tenant sees no
  rows and cannot insert — fail-closed. `DropRLS` reverses it; both idempotent.

- **Enforce** — open the Storage with `Options.EnableRLS`. The pool then sets
  `app.tenant_id` per checked-out connection from `storage.WithTenant(ctx, …)`
  (via pgxpool `PrepareConn`) and `RESET`s it on release, so every operation is
  scoped to its tenant with no per-call wiring. The server's tenant middleware
  populates the context from `X-Tenant-ID`; enable the whole thing with
  `serve --rls` (or `BLUNDERDB_RLS=true`).

- **Non-superuser** — PostgreSQL **superusers bypass RLS even with FORCE**, so
  the application must connect as a non-superuser role for RLS to take effect.

- **Which tables** — `rlsTables` in `rls_postgres.go` lists the
  tenant-scoped tables (every table with a `tenant_id` column, `session_state`
  included since schema 2.17.0); `purgeOrder` in `purge_postgres.go` must stay
  a permutation of it. Both are checked against the schema, not against each
  other: `TestPurgeOrderMatchesRLSTables` reads the `tenant_id` tables from the
  embedded migrations, `TestTenantTablesSchemaGuards` checks on a live database
  that `ApplyRLS` polices each of them, and `TestPurgeTenant` counts what a
  purge leaves behind (#363). `metadata` and
  `schema_migrations` are database infrastructure with no `tenant_id` and no
  per-tenant data, and stay outside — a per-tenant derivative such as the
  cached blunder count lives in `library_settings` (migration 048), under the
  policy and purged with the tenant. The triggers that invalidate it run as
  the writing session, so the policy confines their `DELETE` to the tenant
  whose rows the statement wrote. A forward migration that adds a
  tenant-scoped table installs the policy itself when the database already
  enforces RLS (see `migrations/013_session_state.sql`), so no table is left
  unprotected between the migration and the next `ApplyRLS`.

Application-level tenant filtering stays in place either way — RLS is
belt-and-suspenders, not a replacement.

## PostgreSQL server tuning

`PRAGMA`s have no PostgreSQL equivalent; the SQLite `applyPragmas` step does
not run for this backend. Performance tuning lives in `postgresql.conf` and is
an operator responsibility — the application hardcodes nothing. Recommended
starting points for a dedicated server:

| Parameter              | Suggested starting point          |
|------------------------|-----------------------------------|
| `shared_buffers`       | 25 % of system RAM                |
| `effective_cache_size` | 50–75 % of system RAM             |
| `work_mem`             | 16–64 MB (raise for heavy stats)  |
| `max_connections`      | ≥ pool `MaxConns` × app instances |
| `synchronous_commit`   | `on` (default; `off` trades durability for throughput) |

The connection pool (`pgxpool`) is sized from `BLUNDERDB_POSTGRES_MAX_CONNS`
(default 50). Ensure `max_connections` comfortably exceeds the sum of pool
sizes across all application instances.

## The loader role — offline bulk load of one tenant's library

`cmd/blunderdb-load` (package `bulkload`) loads a large collection of `.xg`
files into one tenant, given by `--tenant N` (above 0), as `blunderdb_loader`. The role is created by the
exploitant (superuser), `NOSUPERUSER NOBYPASSRLS`, `NOLOGIN` outside a load;
the engine never creates it. When it exists and the server runs with
`BLUNDERDB_LOADER_TENANT=N` (unset or invalid, logged: the role reaches nothing, and a start revokes the policy and rights an earlier start gave it), `ApplyRLS` (run by the tables'
owner at each `serve --rls` start) grants it `SELECT, INSERT, UPDATE, DELETE`
on the tenant tables (never `TRUNCATE`, which no policy filters), `SELECT` on
`metadata` and `schema_migrations`, the sequences, and adds on every tenant
table:

```sql
CREATE POLICY loader_public_only ON position AS RESTRICTIVE FOR ALL
    TO blunderdb_loader USING (tenant_id = N) WITH CHECK (tenant_id = N);
```

A restrictive policy is AND-ed with `tenant_isolation`: whatever
`app.tenant_id` the session sets, it reaches tenant N only. Changing the variable replaces the policy at the next start; `OpenLoader` refuses a `--tenant` the policy does not name.

- **Why not `BYPASSRLS`.** `COPY FROM` is refused on any table where RLS
  applies to the role, a permissive policy of its own included; only
  `BYPASSRLS` lifts that, and it opens every tenant. The loader sends its rows
  by `COPY` to a **session temporary table** (no policy, no lasting object),
  then the ordinary `INSERT … SELECT … ON CONFLICT … RETURNING` moves them under
  the policy. Only a loader's transaction does so (`copyTx`); the pool and a
  server's transactions keep the `unnest` statements. `COPY` straight into `position` could not do the conflict on
  `(tenant_id, zobrist_hash)` nor return ids anyway.
- **Never serves.** `postgres.Open` (every server and migration path) refuses
  a role that is, or is a member of, the loader; the loader opens with
  `OpenLoader`, which does not migrate and refuses a schema behind this binary.
- **Secret.** `--password-file` (default `/run/secrets/blunderdb_loader_password`);
  a `--dsn` holding a password is refused; no fallback password.
- **Resumable.** One import batch per load: each group of files commits with
  its journal lines, and `--batch N` skips by SHA-256 what committed.
  Positions, matches and analyses are the ordinary import's (test
  `TestLoadMatchesTheOrdinaryImport`). The public journal is cut
  (`change_journal = off`) and never restored by the load. No index is dropped:
  that needs the table owner, and the tables hold every tenant.
