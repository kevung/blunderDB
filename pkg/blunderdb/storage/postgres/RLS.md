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
