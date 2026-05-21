# P3 — PostgreSQL backend

**Goal.** Implement `storage/postgres/` satisfying the `Storage` interface
from [P2](02-storage-interface.md), with multi-tenant data isolation via a
`tenant_id BIGINT NOT NULL` column on every domain table.

**Estimate.** 10-15 days. **Risk.** High (≈10 k lines of SQL ported).
**PRs.** 8 (split by family).

**Prerequisites.** [P2](02-storage-interface.md).

## Driver

`github.com/jackc/pgx/v5` with `pgxpool`. Rationale:
- Native, idiomatic Go (no CGO, like `modernc.org/sqlite`).
- Built-in connection pool with limits.
- First-class `BYTEA` support.
- Already widely deployed; battle-tested.

Add to `go.mod`:
```
require github.com/jackc/pgx/v5 v5.x.x
```

## Multi-tenant strategy

**Decision (acted): a `tenant_id BIGINT NOT NULL` column on every domain
table.** Single shared schema. Per-tenant Zobrist dedup via composite unique
index `(tenant_id, zobrist_hash)`.

Alternatives considered and rejected for this iteration:
- **Schema-per-tenant** — operationally painful at 10 k tenants
  (catalog bloat, migration × N).
- **Row-Level Security only** — relies on `SET LOCAL role` per request;
  document as a future option, not a hard requirement.

**RLS as optional defence-in-depth.** A migration adds RLS policies on each
table that filter by a `current_setting('app.tenant_id')::bigint`. When
enabled, the application must `SET LOCAL app.tenant_id = $1` at
transaction start. Documented in
`pkg/blunderdb/storage/postgres/RLS.md`; disabled by default — application
filtering is mandatory regardless.

## Schema strategy

**Initial schema = v2.7.0 (terminal state of the SQLite chain).** We do
**not** port the 15 historical SQLite migrations. PostgreSQL starts fresh
at v2.7.0 and tracks its own forward chain from there
(`002_*.sql` is reserved for [P4](04-session-scope.md)'s
`scope` column).

Documented explicitly in
`pkg/blunderdb/storage/postgres/migrations/README.md`.

```
pkg/blunderdb/storage/postgres/migrations/
  001_initial_v2_7_0.sql
  README.md
```

## Type conversions

| SQLite                                | PostgreSQL                              |
|---------------------------------------|------------------------------------------|
| `INTEGER PRIMARY KEY AUTOINCREMENT`   | `BIGSERIAL PRIMARY KEY`                  |
| `INTEGER` (signed 64)                 | `BIGINT`                                 |
| `TEXT`                                | `TEXT`                                   |
| `BLOB`                                | `BYTEA`                                  |
| `DATETIME DEFAULT CURRENT_TIMESTAMP`  | `TIMESTAMPTZ DEFAULT now()`              |
| `REAL`                                | `DOUBLE PRECISION`                       |
| `INSERT OR REPLACE INTO t (k,v) …`    | `INSERT … ON CONFLICT (k) DO UPDATE SET v = EXCLUDED.v` |
| `INSERT OR IGNORE INTO t …`           | `INSERT … ON CONFLICT DO NOTHING`        |
| `result.LastInsertId()`               | `… RETURNING id` + `Scan`                |
| `PRAGMA foreign_keys = ON`            | implicit; `ON DELETE CASCADE` clauses kept |
| `PRAGMA WAL / cache_size / mmap_size` | tuned in `postgresql.conf` (outside repo)|

## Pool configuration

```go
cfg, _ := pgxpool.ParseConfig(dsn)
cfg.MaxConns = 50              // default; configurable via env
cfg.MinConns = 5
cfg.MaxConnLifetime = time.Hour
cfg.HealthCheckPeriod = 30 * time.Second
```

Exposed via `BLUNDERDB_POSTGRES_MAX_CONNS` env var.

## Backend selection

`storage.Open(ctx, dsn, opts)` dispatches on the DSN scheme:
- `sqlite:///path/to/file.db` or `:memory:` → `sqlite.Open`
- `postgres://user:pass@host:port/db?sslmode=...` → `postgres.Open`

In `cmd/blunderdb`:
```go
backend := os.Getenv("BLUNDERDB_BACKEND")  // "sqlite" (default) | "postgres"
dsn     := os.Getenv("BLUNDERDB_DSN")
s, err  := storage.Open(ctx, dsn, opts)
```

The `serve` subcommand ([P6](06-serve-http.md)) reads `--backend` and
`--dsn` flags that override the env.

## Tests via `testcontainers-go`

```go
// pkg/blunderdb/storage/postgres/postgres_test.go
func TestContract_Postgres(t *testing.T) {
    ctx := context.Background()
    container, _ := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:16-alpine"),
        postgres.WithDatabase("blunderdb"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
    )
    defer container.Terminate(ctx)
    dsn, _ := container.ConnectionString(ctx, "sslmode=disable")
    s, _ := postgrespkg.Open(ctx, dsn, nil)
    defer s.Close()
    _ = s.Migrate(ctx)
    storage.RunContractTests(t, func() storage.Storage { return s })
}
```

CI must allow Docker for the `testcontainers` step. If the existing CI
runners disallow it, run Postgres tests behind a build tag
(`//go:build postgres`) and a dedicated job.

Add to `go.mod`:
```
require github.com/testcontainers/testcontainers-go/modules/postgres v0.x
```
(test-only; tracked separately if vendor policy applies.)

## Splitting into 8 PRs

| PR | Scope |
|---|---|
| 1 | **Done.** Skeleton: `postgres.go` (Open, Close, `pgxpool`), `migrations/001_initial_v2_7_0.sql` + `README.md`, `schema_postgres.go` bootstrap, `tx_postgres.go`, `RLS.md`, `doc.go`, all 14 family stores stubbed (return a `not implemented` error wrapping `storage.ErrInternal`). Migration test (`postgres_test.go`, `//go:build postgres`) provisions PostgreSQL 16 via `testcontainers-go` and verifies the 16 tables, 23 indexes, `tenant_id` columns and the `database_version` row. The full contract suite is wired in PR2 once positions/analyses land. |
| 2 | Positions + analyses. Unique index `(tenant_id, zobrist_hash)`. Denormalized columns. |
| 3 | Matches + games + moves + move_analyses (cascade behaviour). |
| 4 | Tournaments + collections + collection_position. |
| 5 | Comments + anki_deck + anki_card. |
| 6 | Filters + session + command_history + search_history. (Note: `scope` not yet — depends on P4.) |
| 7 | Stats + metadata. Statistical SQL ported (largest body — `db_stats.go` ≈ 1232 L). |
| 8 | RLS migration (optional opt-in), tenant isolation tests, `concurrent_writes_test.go`. |

## Tests to write (beyond contract)

- `tenant_isolation_test.go`: insert position for tenant=1 and tenant=2,
  verify `Positions().List(scope=tenant1)` returns only tenant 1's row,
  same for tenant 2. Same Zobrist hash → 2 rows (one per tenant), not 1.
- `concurrent_writes_test.go`: 100 goroutines × 1 000 inserts each across
  10 tenants. Final row count exact. No errors.
- `migration_test.go`: fresh DB, run `Migrate(ctx)`, confirm all 16 tables
  + expected indexes exist; `database_version` row = "2.7.0" (or current).
- `large_dataset_bench_test.go` (deferred to [P9](09-benchmarks.md), but
  the scaffolding can land here): 1 M positions, query latency.

## Gotchas

1. **`zobrist_hash` is `int64` signed in Go** but represents a 64-bit hash.
   `BIGINT` in Postgres is signed 64-bit. Storing the bit pattern as a
   signed integer is fine; just be consistent between writes and reads.
2. **`GROUP_CONCAT`** (used in some SQLite migration code in
   `db_migration.go:288`) does not exist in Postgres. Since we do not port
   historical migrations, this is moot — but watch for any runtime SQL
   that uses it.
3. **`PRAGMA`s have no Postgres equivalent.** `applyPragmas` only runs on
   SQLite. Document in `RLS.md` which Postgres parameters are recommended
   (`shared_buffers`, `effective_cache_size`, `max_connections`,
   `work_mem`, …) but do not hardcode anything in the application.
4. **`DELETE … CASCADE`** is declared on FKs in SQLite via `ON DELETE
   CASCADE`. Make sure every FK in the Postgres schema explicitly declares
   `ON DELETE CASCADE` where the SQLite equivalent does.
5. **`bool` representation.** SQLite uses `INTEGER 0/1`; Postgres has
   native `BOOLEAN`. Adjust columns like `is_forced`, `is_close_cube`,
   `no_contact` accordingly. The Go types stay `bool`.
6. **JSON columns.** Today the `analysis.data` blob is zlib-compressed
   bespoke binary (not JSON). Keep it `BYTEA`. The `metadata` table's
   `value` is `TEXT` (some entries hold JSON strings). Keep `TEXT` to avoid
   over-engineering; we can switch to `JSONB` later if querying needs it.
7. **Driver-specific transaction isolation.** Default Postgres isolation is
   `READ COMMITTED`. Loose stats queries are fine at this level; document
   in `pkg/blunderdb/storage/postgres/doc.go`.

## Verification

- [ ] Contract test green on Postgres via `testcontainers-go`.
- [ ] `BLUNDERDB_BACKEND=postgres BLUNDERDB_DSN=... ./blunderdb info`
      returns `database_version` — confirms the CLI works with Postgres.
- [ ] `tenant_isolation_test.go` green.
- [ ] `concurrent_writes_test.go` green at 100 goroutines × 1 000 inserts
      without errors.
- [ ] All `Storage` methods implemented (`grep ErrNotImplemented
      pkg/blunderdb/storage/postgres/` empty).

## Risks

- Volume of SQL to port. Mitigation: split into 8 small PRs, contract
  test catches drift.
- Slight behavioural differences (e.g. ordering without `ORDER BY` is
  undefined in both, but `INSERT … RETURNING id` semantics, `now()`
  precision differ). The contract test should not assume an exact ordering
  unless an `ORDER BY` is in the query.
- CI Docker availability — fallback plan: run Postgres tests in a
  dedicated job behind `//go:build postgres`.
