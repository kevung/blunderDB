# P8 — Streaming imports + `context.Context` cancellation

**Goal.** Replace the global `importCancelled` atomic flag with proper
`context.Context` cancellation propagated through every import path, and
support per-request cancellation in the HTTP API. Bonus: stream uploads
via `multipart/form-data` instead of buffering whole files in RAM.

**Estimate.** 3-5 days. **Risk.** Medium. **PRs.** 1 (possibly 2).

**Prerequisites.** [P2](02-storage-interface.md) (`ctx` in interface),
[P6](06-serve-http.md) (`/v1/imports.cancel` endpoint).

## Why

- Today, `importCancelled int32` is a process-global atomic. A second
  import cannot run in parallel without confusing the flag.
- In server mode, every request has its own `context.Context` from
  `r.Context()`. Per-request cancellation requires plumbing `ctx` through
  the importers.
- A user closing their HTTP connection mid-upload should abort the import
  on the server within ~100 ms.

## Steps

- [ ] **Add `ctx context.Context`** to every method in the importer
      packages (`pkg/blunderdb/importers/*.go`). They already take it from
      `Storage` in P2; ensure they propagate it down to every loop
      iteration that today checks `atomic.LoadInt32(&d.importCancelled)`.
- [ ] **Replace the atomic check** at every callsite:
      ```go
      // before
      if atomic.LoadInt32(&d.importCancelled) != 0 {
          return ErrImportCancelled
      }
      // after
      if err := ctx.Err(); err != nil {
          return err
      }
      ```
      Granularity: at every match boundary (current behaviour), at every
      DB write inside a long batch, and at every parser progress callback.
- [ ] **Remove `importCancelled` field** from `Database` (and from
      anything else that referenced it).
- [ ] **`defer tx.Rollback()` everywhere.** Even when `ctx.Err()` aborts a
      partial import, the transaction must be rolled back or the
      connection is left holding locks. Add a panic-recover around each
      import body to ensure rollback even on unexpected error paths.
- [ ] **Per-request cancellation in the server** (P6 surfaced this):
      ```go
      // internal/server/handlers/imports.go
      type Server struct {
          imports map[string]context.CancelFunc   // key: importID
          mu      sync.Mutex
      }
      func (s *Server) startImport(...) string {
          id := uuid.NewString()
          ctx, cancel := context.WithCancel(parentCtx)
          s.mu.Lock(); s.imports[id] = cancel; s.mu.Unlock()
          go s.runImport(ctx, id, ...)
          return id
      }
      func (s *Server) cancelImport(id string) {
          s.mu.Lock(); defer s.mu.Unlock()
          if c, ok := s.imports[id]; ok { c(); delete(s.imports, id) }
      }
      ```
- [ ] **Streaming upload** via `multipart/form-data`:
      ```go
      reader, err := r.MultipartReader()
      for {
          part, err := reader.NextPart()
          if err == io.EOF { break }
          if part.FormName() == "file" {
              importer.ParseStream(ctx, part)   // consumes io.Reader directly
          }
      }
      ```
      External parsers (`xgparser`, `gnubgparser`, `bgfparser`) currently
      take a `filePath`. Two paths:
      - **Easy**: write the upload to a temp file, then call the existing
        `Parse(filePath)`. Acceptable since file sizes are small.
      - **Better**: extend the parser libraries (which live in separate
        repos under `github.com/kevung/…`) to accept `io.Reader`. Defer
        this until volumes justify it.
      Recommendation: temp-file path for this iteration; track upstream
      work as a follow-up.
- [ ] **Progress events**. While importing, emit NDJSON events to the
      response writer:
      ```json
      {"event":"progress","matches":1,"games":3,"positions":120}
      ```
      Backed by the existing `migrationProgress` callback shape, re-used
      for imports.

## Gotchas

1. **`importCancelled` had ≈ 10 callsites** scattered across importers
   and possibly migration code. Audit:
   ```bash
   grep -rn 'importCancelled' --include='*.go'
   ```
2. **Migrations.** If `db_migration.go` references `importCancelled` (it
   shouldn't — migrations are not cancellable mid-flight typically),
   replace with a no-op or with `ctx.Err()` — but a migration that
   half-runs is dangerous. Either keep migrations non-cancellable
   (preferred: they are quick) or wrap the whole migration in a single
   transaction so a rollback restores state.
3. **HTTP server shutdown**. When the daemon is stopped (SIGTERM), the
   root context is cancelled, which cancels all active imports. Each
   import must complete its `tx.Rollback()` within the grace period
   (`http.Server.Shutdown` timeout). Document a recommended timeout.
4. **Cancellation map leaks**. If a successful import does not remove
   itself from `s.imports`, the map grows. Always `defer
   s.removeImport(id)` regardless of outcome.
5. **Concurrent imports per tenant**. The spec does not forbid them.
   Each gets its own `importID`. The cancellation map handles concurrency.

## Tests

- [ ] `cancellation_test.go`: start an import, call cancel after 50 ms,
      assert (a) the import returns within 100 ms, (b) no row is committed,
      (c) the transaction is rolled back.
- [ ] `concurrent_imports_test.go`: two tenants import in parallel; both
      succeed; cancel one, the other completes.
- [ ] `shutdown_test.go`: `server.Shutdown(ctx)` aborts a running import
      cleanly.
- [ ] `multipart_upload_test.go`: stream a 100 MB file through
      `multipart`; peak RSS stays bounded.

## Verification

- [ ] `grep -rn 'importCancelled' pkg/` returns no hits.
- [ ] `POST /v1/imports.xg` followed by `POST /v1/imports.cancel` triggers
      a clean abort within 200 ms.
- [ ] `server_test.go` cancellation case green.
- [ ] All previous tests green.

## Risks

- Subtle issue: the parser libraries may not check `ctx` mid-parse (they
  are external). In practice, a single match parses in <100 ms, so granular
  cancellation between matches is acceptable.
- Long-running stats queries (not imports, but similar shape) should
  also honour `ctx.Done()`. Audit during this phase but do not expand
  scope unless trivial.

## PR layout

Single PR: `feat(server): plumb context cancellation through imports;
remove global importCancelled flag`.
