# Row-Level Security & PostgreSQL tuning

## RLS — optional defence-in-depth

Multi-tenant isolation in blunderDB is **enforced by the application**: every
query filters by `tenant_id`. This is mandatory and sufficient.

Row-Level Security is offered as an optional second layer. It is **not** part
of `001_initial_v2_7_0.sql` and is **disabled by default**. A later migration
(landed in P3 PR8) adds, per domain table:

```sql
ALTER TABLE position ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON position
    USING (tenant_id = current_setting('app.tenant_id')::bigint);
```

When RLS is enabled, the application must issue
`SET LOCAL app.tenant_id = $1` at the start of every transaction. Application
filtering stays in place either way — RLS is belt-and-suspenders, not a
replacement.

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
