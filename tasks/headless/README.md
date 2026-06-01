# Headless Mode — Task Index

This folder tracks the multi-phase refactor that turns blunderDB into a
**generic headless engine** consumable by external services, while preserving
the existing desktop GUI and CLI binaries.

> **Scope reminder.** blunderDB stays a generic open-source backgammon engine.
> No specific consumer or product is referenced in this repo. The `serve` mode
> is a legitimate operational mode (alongside `gui` and `cli`), not tied to
> any particular deployment.

## Goal

Allow blunderDB to be used as:
1. **A Go library** (`import "github.com/kevung/blunderdb/pkg/blunderdb/..."`).
2. **A network daemon** (`blunderdb serve`) exposing the engine over HTTP+JSON
   with optional multi-tenant isolation, scalable to 10 000+ concurrent users
   when backed by PostgreSQL.

The desktop binary (`wails build`) and the existing CLI subcommands keep
working identically throughout the chantier — no big-bang refactor.

## Decisions actées

| # | Decision |
|---|---|
| 1 | **Pluggable storage** via a `Storage` interface. SQLite stays the default backend (GUI/CLI/desktop). PostgreSQL is an additional backend for the server mode. |
| 2 | **Daemon protocol = HTTP + JSON** (no gRPC). RPC-style routes: `POST /v1/<family>.<method>`. |
| 3 | **Authentication is delegated** to an upstream reverse-proxy. The daemon trusts the `X-Tenant-ID` request header. The daemon must not be exposed directly to the public internet. |
| 4 | **Library refactor**: Go code moves from `package main` to `pkg/blunderdb/...`. Module path becomes `github.com/kevung/blunderdb`. |
| 5 | **Multi-tenant model** (PostgreSQL): a `tenant_id BIGINT NOT NULL` column on every domain table. Per-tenant Zobrist dedup. Row-Level Security is optional, documented but off by default. |
| 6 | **Backward compatibility**: existing `wails build`/`wails dev` and the historical CLI subcommands must keep working through every phase. |
| 7 | **Branch**: `feature/headless`. Phases land as series of small PRs (see each task sheet). |

## DAG of phases

```
P0 module-rename ──► P1 pkg/ refactor
                       │
                       ▼
P2 Storage interface ──► P3 Postgres backend
   │                        │
   ▼                        │
P4 session-scope ◄──────────┤
   │                        │
   ▼                        ▼
P5 remove-mutex ─────────► P6 serve HTTP
                              │
                              ├──► P7 CLI 100 %
                              └──► P8 streaming + ctx
                                        │
                                        ▼
                                   P9 benchmark 10k
```

Optional side phases:
- **Phase A** between P1 and P2: split `cli.go` (2241 L) into one file per
  subcommand under `internal/cli/`.
- **Phase B** after P3: `blunderdb migrate sqlite→postgres` tool.
- **Phase C** after P6: per-tenant rate-limiting middleware.

## Task sheets

| # | Phase | Sheet | Estimate | Risk | Status |
|---|---|---|---|---|---|
| P0 | Module rename | [00-module-rename.md](00-module-rename.md) | ½ d | low | **done** |
| A | CLI extract | [01a-cli-extract.md](01a-cli-extract.md) | 1-2 d | low | **done** (4743e21d) |
| P1 | Library refactor | [01b-pkg-library-refactor.md](01b-pkg-library-refactor.md) | 5-7 d | high | **done** (PR1 domain+engine, PR2 database, PR3 cmd/internal) |
| P2 | `Storage` interface | [02-storage-interface.md](02-storage-interface.md) | 8-10 d | high | PR1 done (`c8bcfc1e`); design arbitrated (D1-D10); PR2 done (SQLite positions/analyses/search + wrapper delegation + contract tests); PR3 done (SQLite matches + tournaments, D2 applied, contract Match/* enabled); PR4 done (SQLite collections + comments + anki, contract Collection/* enabled); PR5 done (SQLite filters + session + history + metadata, D6 + D10 applied); PR6 pending |
| P3 | Postgres backend | [03-postgres-backend.md](03-postgres-backend.md) | 10-15 d | high | PR1-5 done (positions/analyses/search, matches/games/moves, tournaments/collections, comments/anki); **PR6 done** (filters + command_history + search_history + metadata incl. tenant-scoped `Counts`). **Stats done on SQLite** — `statsStore` ported from `database/db_stats.go` into `storage/sqlite/stats_sqlite.go` (all 7 methods), gated by a JSON parity test vs the legacy `Database` over 4 XG fixtures; `Stats/AggregateCounts` contract case enabled (auto-skips on backends that return not-implemented). **Remaining:** Postgres **stats** (needs per-tenant scoping of every aggregate query — still a stub) + Postgres **session** (now unblocked: P4 added tenant-scoped metadata keys) + RLS migration |
| P4 | Session scope | [04-session-scope.md](04-session-scope.md) | 3-4 d | medium | **session done** — session state is namespaced per scope by prefixing its `metadata` keys (`sessionScopedKey`): empty scope (GUI/CLI) unchanged → no migration/version bump/regression; daemon tenants isolated. Both backends; contract `Session/SaveLoadEmpty` + `Session/MultiScopeIsolation` green. **Deferred:** SQLite `scope` columns for filter/command/search history (Postgres already scopes them via `tenant_id`; only matters for multi-tenant SQLite-as-server) |
| P5 | Remove global mutex | [05-remove-global-mutex.md](05-remove-global-mutex.md) | 4-6 d | high | **PR1 done** (re-scoped) — server path was already mutex-free via `storage.Storage`; PR1 added per-connection `busy_timeout`+pragmas via `_pragma` DSN, pool sizing (`:memory:`=1 conn, file=10), and `-race`-green concurrency tests on the storage layer. PR2/PR3 (strip the 240 `d.mu.*` in the GUI-only `Database` wrapper) **deferred** — single-user GUI, no scaling benefit, regression risk |
| P6 | `serve` HTTP+JSON | [06-serve-http.md](06-serve-http.md) | 5-7 d | medium | **done** — PR1 (skeleton + middlewares + health/metrics + frozen error envelope + `serve` subcommand), PR2 (100 `/v1/<family>.<method>` handlers across all 14 families), PR3 (imports/exports over Storage — see sheet 12) |
| P7 | CLI 100 % coverage | [07-cli-full-coverage.md](07-cli-full-coverage.md) | 3-4 d | low | **done** — `blunderdb call <family>.<method>` dispatches **in-process** through the live server handler (`Server.Handler()` + `Server.Paths()`), so all 108 routes share the exact handler functions (CLI/HTTP parity) with no route-table extraction. `--list`, `--json`/`--json-file`, `--scope`, sqlite/postgres. Legacy subcommands untouched |
| P8 | Streaming + ctx | [08-streaming-imports-ctx.md](08-streaming-imports-ctx.md) | 3-5 d | medium | pending |
| P9 | Benchmarks | [09-benchmarks.md](09-benchmarks.md) | 3-4 d | low | pending |
| B | SQLite→Postgres tool | [10-sqlite-to-postgres-tool.md](10-sqlite-to-postgres-tool.md) | 2-3 d | medium | **core done** — `blunderdb migrate --from sqlite:// --to postgres:// --tenant-id` copies positions/analyses/comments/matches(+games+moves)/tournaments/collections via the Storage interface with FK remap, in one dest tx; `--dry-run`, `--on-conflict skip`, conflict guard; NDJSON progress. `pkg/blunderdb/migrate`, postgres-tagged testcontainers test. **Deferred:** anki/filters/history/session (P4-dependent app-state) |
| C | Per-tenant rate-limit | [11-tenant-rate-limit.md](11-tenant-rate-limit.md) | 1-2 d | low | **done** — dependency-free per-tenant token bucket (`middleware.RateLimit`/`RateLimiter`, injectable clock, idle-bucket sweeper); opt-in via `serve --rate-limit-rps/--rate-limit-burst`; `rate_limited` (429) envelope; `blunderdb_ratelimit_{rejected_total,buckets}` metrics |
| P6.3/P8 | Imports/exports over Storage | [12-imports-exports-over-storage.md](12-imports-exports-over-storage.md) | 6-9 d | high | **done** — PR3a (ingest interfaces + HTTP transport + JSON interchange); PR3b (XG mapper + parity gate); PR3c (GnuBG SGF/MAT, BGF, native .db, XGP + BGF-text positions) + cross-format canonical-duplicate enrichment. All routes wired & parity-gated: imports.{json,xg,gnubg,bgf,db,position} |

Conventions and shared vocabulary: see [glossary.md](glossary.md).

## Points de vigilance (cross-phase)

1. **`Database` wrapper = accepted technical debt.** The Wails frontend
   binds against `Database`'s public method set (auto-generated under
   `frontend/wailsjs/go/main/`). The wrapper must stay until the GUI itself
   migrates to HTTP. Document in `pkg/blunderdb/database/doc.go`.
2. **SQLite-as-server ≠ 10 k users.** Even with WAL + `busy_timeout`, SQLite
   tops out around 1-2 k ops/s under contention. SQLite-server is fine for
   small deployments (≤ 100 active users). PostgreSQL is the path to 10 k.
3. **HTTP error format frozen in P6.** External clients depend on the shape.
   Picked once, never broken: `{"error":{"code":"...","message":"..."}}`
   with a closed code set (`not_found`, `conflict`, `invalid`, `internal`).
4. **No external tool/SaaS/consumer reference** anywhere in this repo. The
   server mode is described as a generic operational mode.

## Verification end-to-end (after P9)

1. `wails build` + `wails dev` — desktop GUI behaves identically.
2. `go test ./...` + `go test ./tests/...` — all tests green, including new
   `contract_test`, `tenant_isolation`, `concurrent_writes`,
   `migration_test 2.7→2.8`, `cancellation_test`.
3. Library usable from a third-party Go program:
   ```go
   import "github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
   s, _ := sqlite.Open(ctx, ":memory:")
   s.Positions().Save(ctx, "", pos)
   ```
4. SQLite server: `./blunderdb serve --backend sqlite --db /tmp/x.db --addr :8080`
   responds; two tenants stay isolated via `X-Tenant-ID`.
5. Postgres server: same with `--backend postgres --dsn ...`;
   `testcontainers-go` provisions Postgres in CI.
6. Bench: `make bench` produces `perf-baseline.md` showing the target 10 k
   RPS sustained on Postgres (pool=50), p95 < 50 ms, p99 < 200 ms.
   (Targets confirmed empirically and adjusted in the report.)
7. CLI 100 %: `blunderdb call <family>.<method>` for every method on `Storage`.
8. Cancellation: import started then HTTP `DELETE` → full rollback in < 200 ms.
