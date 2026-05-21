# P6 — `blunderdb serve` HTTP + JSON daemon

**Goal.** New subcommand `blunderdb serve` that exposes the engine over
HTTP with a JSON RPC-style API, suitable for consumption by an external
service through an upstream reverse-proxy.

**Estimate.** 5-7 days. **Risk.** Medium. **PRs.** 3.

**Prerequisites.** [P2](02-storage-interface.md). Recommended after
[P3](03-postgres-backend.md), [P4](04-session-scope.md),
[P5](05-remove-global-mutex.md).

## API style: RPC, not REST

Routes: `POST /v1/<family>.<method>`.

- The methods on `Storage` are not all CRUD (e.g. `MergePlayers`,
  `SwapMatchPlayers`, `SearchPositions` with complex filters,
  `ReviewAnkiCard`).
- RPC maps 1-for-1 onto the Go interface, keeping handler code mechanical
  (and code-generatable).
- All requests are `POST` with a JSON body; all responses are JSON. List
  endpoints stream NDJSON.

Examples:
```
POST /v1/positions.save        body: { "position": {...} }
POST /v1/positions.list        body: { "filter": {...} }   → NDJSON stream
POST /v1/matches.delete        body: { "id": 42 }
POST /v1/search.byFilter       body: { "filter": "..." }   → NDJSON stream
POST /v1/imports.xg            multipart: file=...         → progress events
POST /v1/imports.cancel        body: { "import_id": "..." }
```

Health and ops endpoints (no `/v1`):
```
GET  /healthz                  → 200 if storage SELECT 1 succeeds
GET  /readyz                   → 200 if healthz + version matches expected
GET  /metrics                  → Prometheus exposition (text format)
```

## Authentication & tenancy

**No authentication in this daemon.** Authentication is delegated to the
upstream reverse-proxy. The daemon trusts the `X-Tenant-ID` header.

Middleware `internal/server/middleware/tenant.go`:
- Extracts `X-Tenant-ID` from the request header.
- Returns `400 invalid` if missing or empty (except on `/healthz`,
  `/readyz`, `/metrics`).
- Puts the tenant ID in the request `context.Context` via a typed key.

The daemon **must not** be exposed directly to the public internet. This is
documented prominently in `serve --help` and in `pkg/blunderdb/api/doc.go`.

## Target structure

```
internal/server/
  server.go              ← New(opts) *Server; Run(ctx) error
  options.go             ← Addr, Storage, Logger, MetricsRegistry, …
  middleware/
    tenant.go            ← X-Tenant-ID extraction
    logging.go           ← structured request log
    metrics.go           ← Prometheus latency / status histograms
    recover.go           ← panic → 500
    cors.go              ← off by default; configurable
  handlers/
    positions.go
    analyses.go
    matches.go
    collections.go
    tournaments.go
    anki.go
    filters.go
    session.go
    search.go
    stats.go
    history.go
    metadata.go
    imports.go
    exports.go
    health.go
  routes.go              ← table: method+path → handler
  ndjson.go              ← streaming helper
  errors.go              ← typed HTTP error envelope
  server_test.go         ← httptest-based integration tests

pkg/blunderdb/api/
  routes.go              ← shared route table (also used by CLI in P7)
  errors.go              ← shared error codes
  doc.go                 ← API documentation (used by godoc)
```

## Error envelope (frozen)

```json
{
  "error": {
    "code": "not_found" | "conflict" | "invalid" | "internal",
    "message": "human-readable description",
    "details": { /* optional, code-specific */ }
  }
}
```

Closed code set. New codes require a major version bump of the API.

Mapping `storage.Err*` → HTTP status:
- `ErrNotFound` → 404 `not_found`
- `ErrConflict` → 409 `conflict` (e.g. Zobrist dedup)
- `ErrInvalid` → 400 `invalid`
- anything else → 500 `internal`

## HTTP framework

`net/http` from the standard library (Go 1.25 supports `{id}` in patterns
natively). **No new HTTP framework dependency.** This keeps the open-source
surface lean and avoids tying users to a specific ecosystem.

## Handler generation

≈ 186 methods → manual handler writing is ~3 days of tedious boilerplate.
**Recommended**: `go generate` reading the `Storage` interface via Go
reflection or AST and emitting handler stubs.

```
go generate ./pkg/blunderdb/api/...
```

Output: `internal/server/handlers/generated_*.go` files with one handler
per method. Custom handlers (multipart imports, NDJSON streaming) are
written by hand and excluded from generation via a build tag.

If code-gen is deferred, write handlers manually in PR 2 (allow 2 extra
days).

## Streaming list endpoints

For methods that return iterators (`Positions().List`,
`Search().ByFilter`, `Anki().ListCards`, …), use NDJSON:

```
Content-Type: application/x-ndjson

{"id":1,"zobrist_hash":...}
{"id":2,"zobrist_hash":...}
...
```

Handler pattern:
```go
func (h *Handler) ListPositions(w http.ResponseWriter, r *http.Request) {
    scope := tenantFromCtx(r.Context())
    w.Header().Set("Content-Type", "application/x-ndjson")
    enc := json.NewEncoder(w)
    flusher, _ := w.(http.Flusher)
    for p, err := range h.storage.Positions().List(r.Context(), scope, opts) {
        if err != nil { writeError(w, err); return }
        _ = enc.Encode(p)
        if flusher != nil { flusher.Flush() }
    }
}
```

## File uploads (`imports.*`)

`multipart/form-data` with a `file` field. The handler creates an
`io.PipeReader` and passes it to the parser. The whole file is **not**
loaded into RAM.

```
POST /v1/imports.xg                Content-Type: multipart/form-data
                                    file=@game.xg
                                    (optional) parameters=...
→ 200 application/x-ndjson
  {"event":"started","import_id":"…"}
  {"event":"progress","matches":1,"games":12,"positions":340}
  {"event":"done","matches":1,"saved_positions":340}
```

Cancellation: `POST /v1/imports.cancel { "import_id": "..." }` calls the
import's `context.CancelFunc`. Active imports tracked in a per-tenant map
in the `Server` struct.

## Splitting into 3 PRs

| PR | Scope |
|---|---|
| 1 | `Server` skeleton, `Options`, middlewares (tenant, logging, recover, metrics), `/healthz`, `/readyz`, `/metrics`, `routes.go` table, error envelope. No domain handlers yet — wired but stubbed. |
| 2 | Core domain handlers (generated or hand-written): positions, analyses, matches, collections, tournaments, anki, comments, filters, session, search, stats, history, metadata. NDJSON streaming for list endpoints. |
| 3 | Import/export handlers (multipart, streaming), cancellation map, progress events. |

## Configuration

`blunderdb serve` flags:

```
--backend <sqlite|postgres>    [or env BLUNDERDB_BACKEND]
--dsn     <connection string>  [or env BLUNDERDB_DSN]
--addr    <host:port>          [default :8080]
--log-level <debug|info|warn>  [default info]
--metrics                       [default true]
```

## Gotchas

1. **Tenant ID type.** HTTP header is a string. The interface uses
   `scope string` (P4) and Postgres uses `tenant_id BIGINT`. Decision:
   the daemon treats `X-Tenant-ID` as an opaque string. Conversion to
   `BIGINT` for Postgres happens at the bottom of the stack (or the column
   stays `BIGINT` and the daemon `strconv.ParseInt`s — pick one and
   document). Recommendation: **`scope` is always a string at the Go
   level**; the Postgres backend coerces via a typed helper.
2. **JSON encoding of `int64`**. JS clients lose precision above 2^53.
   Encode large IDs as strings (`json:"id,string"`) **only if** an external
   consumer struggles. By default, leave as numbers and document the limit.
3. **Streaming and HTTP/1.1 chunked transfer.** `http.Flusher` must be
   used; otherwise the response is buffered. Test that NDJSON arrives
   incrementally, not all at once.
4. **CORS off by default.** The daemon is internal-only. Add a flag
   `--cors-allow-origin '...'` for non-production setups.
5. **Reject huge JSON bodies.** Configure `http.MaxBytesReader` per
   endpoint to avoid OOM from a malicious tenant.
6. **No connection caching by tenant.** Even with SQLite-per-tenant
   patterns, we are using a single shared `Storage` keyed by `scope`. If
   later we want per-tenant SQLite files, the `Storage` becomes a registry
   — out of scope here.

## Tests

- [ ] `server_test.go`: spin up `httptest.NewServer` backed by an
      in-memory SQLite `Storage`. Hit one endpoint per family, assert
      JSON shape.
- [ ] `tenant_isolation_http_test.go`: two tenants via header, mutually
      invisible reads/writes.
- [ ] `cancellation_test.go`: client `r.Cancel()` mid-request → server
      observes `ctx.Done()` within 100 ms.
- [ ] `ndjson_test.go`: streaming endpoint emits records progressively
      (no buffering).
- [ ] `error_envelope_test.go`: every typed error maps to the documented
      HTTP status and code.

## Verification

- [ ] `./blunderdb serve --backend sqlite --db /tmp/x.db --addr :8080`
      starts; `/healthz` → 200.
- [ ] `./blunderdb serve --backend postgres --dsn '...' --addr :8080`
      starts against a real Postgres.
- [ ] `curl -X POST -H 'X-Tenant-ID: u1' :8080/v1/positions.save -d '{…}'`
      → 200, body contains the new ID.
- [ ] `curl -X POST -H 'X-Tenant-ID: u2' :8080/v1/positions.list -d '{}'`
      → does not return `u1`'s data.
- [ ] `/metrics` returns Prometheus text format.
- [ ] `serve --help` warns "do not expose directly to the public internet;
      put behind a reverse-proxy that handles authentication".

## Documentation

- Update `CLI_USAGE.md` with a `serve` section (or split out `SERVE.md`).
- `pkg/blunderdb/api/doc.go` documents the route table and the error
  envelope.
- An `examples/curl/*.sh` directory with one example per family.

## Risks

- Hand-writing 186 handlers is tedious. Code-gen mitigates but adds a
  generator to maintain.
- API shape locked-in by first external consumer. Be conservative on the
  surface in PR 2; expand later.
