# Headless — Glossary & Conventions

Shared vocabulary for the headless refactor. Keep this short. If a term is
ambiguous in code reviews, define it here first.

## Terms

**Storage.** The Go interface (introduced in [P2](02-storage-interface.md))
that abstracts all persistence operations. Implemented by `storage/sqlite/`
and `storage/postgres/`. Methods take `ctx context.Context` and, when
multi-statement, an explicit `Tx`.

**Database.** The wrapper type (`pkg/blunderdb/database/Database`) preserved
across the refactor for two reasons: (1) Wails auto-binds against its public
method set, and the Svelte frontend depends on those generated bindings;
(2) it gives the GUI/CLI a stable façade independent of which `Storage`
implementation is plugged in.

**scope.** A string attached to "session-like" state rows (`metadata`,
`command_history`, `search_history`, `filter_library`) introduced in
[P4](04-session-scope.md). Empty `""` in single-user mode (GUI/CLI). In
server mode, `scope == tenant_id`.

**tenant_id.** The PostgreSQL column carried on every domain table to
partition data per consumer in [P3](03-postgres-backend.md). Provided to the
server by the upstream reverse-proxy via the `X-Tenant-ID` HTTP header. The
server does **not** validate that the tenant exists — first write creates
the tenant's data space implicitly.

**Library mode.** Using the engine by importing
`github.com/kevung/blunderdb/pkg/blunderdb/...` from a Go program. Maximum
performance, no IPC. Introduced in [P1](01b-pkg-library-refactor.md).

**Server mode.** Running `blunderdb serve` to expose the engine over HTTP.
Introduced in [P6](06-serve-http.md).

**GUI mode / CLI mode.** Existing operational modes — dispatched in `main.go`
on `os.Args[1]`. Unchanged in functionality through the chantier.

**RPC-style API.** Route convention `POST /v1/<family>.<method>` (rather
than REST CRUD). Family = a sub-interface of `Storage` (positions, matches,
analyses, ...). Method = the Go method name in `lowerCamelCase`.

**Contract test.** A parametrised test suite (`storage/contract_test.go`)
that exercises every method of `Storage` against an implementation. Re-used
for both SQLite and PostgreSQL backends to keep them behaviourally aligned.

## File layout conventions

```
pkg/blunderdb/                       ← public library surface
  domain/        types
  engine/        Zobrist, bitboards, EPC (with //go:embed gnubg_os6.bd)
  storage/       Storage interface + sub-interfaces by family
    sqlite/      modernc.org/sqlite implementation
    postgres/    pgx/v5 implementation + migrations/*.sql
  database/      Database wrapper (kept for Wails)
  api/           route table shared by server & CLI dispatcher
  importers/     XG/GnuBG/BGF/native parsers wiring
  migration/     schema-version logic
cmd/
  blunderdb/     main.go (GUI/CLI/serve dispatch)
  blunderdb-loadtest/   load test tool (P9)
internal/        not importable by external programs
  gui/           Wails-only code (app.go)
  cli/          CLI subcommand implementations
  server/       HTTP handlers, middleware, routes
```

## Naming

- Sub-interfaces of `Storage` end in `Store`: `PositionStore`, `MatchStore`,
  `AnkiStore`, etc.
- Methods on `Storage` return `(T, error)` or `error`; never panic.
- `tenant_id` is a `int64` in Go, `BIGINT NOT NULL` in Postgres.
- `scope` is `string` (Go) / `TEXT NOT NULL DEFAULT ''` (SQL).
- HTTP error envelope (frozen in P6):
  ```json
  { "error": { "code": "not_found", "message": "…", "details": {…} } }
  ```
  Closed code set: `not_found`, `conflict`, `invalid`, `internal`.

## Migration discipline

- Every schema change bumps `DatabaseVersion` in `model.go`.
- SQLite migration: add a path in `db_migration.go` (or its package
  equivalent after P1).
- PostgreSQL migration: add a numbered SQL file under
  `pkg/blunderdb/storage/postgres/migrations/NNN_name.sql`.
- Always covered by a test in `migration_test.go` (or successor).
- The two backends do **not** share migration scripts; PostgreSQL starts
  fresh at v2.7.0 and tracks its own forward chain from there.
