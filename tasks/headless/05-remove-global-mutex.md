# P5 — Remove the global `Database.mu` mutex

**Goal.** Replace the broad `sync.RWMutex` that serialises virtually every
DB operation with finer-grained, scoped synchronisation: rely on the
thread-safe `*sql.DB` connection pool, configure `busy_timeout`, and use
explicit transactions for cross-statement invariants.

**Estimate.** 4-6 days. **Risk.** High (race conditions). **PRs.** 3.

**Prerequisites.** [P2](02-storage-interface.md) (interface in place),
ideally [P4](04-session-scope.md).

## Why

- Today every public `Database` method takes `d.mu.Lock()` or
  `d.mu.RLock()`. With ≈ 240 callsites, the lock effectively serializes
  the whole engine, capping throughput at ~100-200 ops/s/instance.
- `*sql.DB` is already concurrency-safe and manages its own pool. Wrapping
  it in a single mutex defeats the pool.
- For server mode ([P6](06-serve-http.md)) to scale to 10 k users, this
  serialisation must go.

## Strategy

Walk every `d.mu.*` call site and classify:

**Class A — Mutex unnecessary.** Single `Exec` or `Query`. SQL driver
already locks its own connection. → Remove mutex.

**Class B — Multi-statement invariant.** Wrap in `BeginTx`/`Commit`. The
DB's locking guarantees atomicity. → Replace mutex with explicit
transaction.

**Class C — Protects a Go field.** Methods that mutate `Database.db`
itself (e.g. `SetupDatabase`, `OpenDatabase` swap the `*sql.DB` pointer
when the user opens a new file). → Keep a fine-grained mutex only on the
field, or refactor to make `Storage` immutable once constructed (new
file → new `Storage` instance) and have the wrapper hold the swap behind
a tiny dedicated lock.

## Configuration changes

### SQLite

In `applyPragmas` (currently in `db.go`):

```go
// Existing:
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = -65536;
PRAGMA temp_store = MEMORY;
PRAGMA mmap_size = 268435456;
PRAGMA foreign_keys = ON;

// Add:
PRAGMA busy_timeout = 5000;          -- 5 s before SQLITE_BUSY
```

Connection pool sizing on the `*sql.DB`:

```go
db.SetMaxOpenConns(10)     // for SQLite-server mode
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(time.Hour)
```

For the in-memory mode used by the GUI, keep `MaxOpenConns = 1` — `:memory:`
databases are per-connection.

### PostgreSQL

`pgxpool` is already configured in P3 with `MaxConns = 50`. No further
change here; the global mutex never applied to Postgres anyway.

## Audit method

```bash
grep -rn 'd\.mu\.' --include='*.go' > /tmp/mutex-audit.txt
# Or post-P1: grep -rn '\.mu\.' pkg/blunderdb/ > /tmp/mutex-audit.txt
```

Walk each line and tag with `A`, `B`, or `C`. Track in a checklist. Aim for
zero Class A locks and explicit `Tx` for Class B.

## Splitting into 3 PRs

| PR | Scope |
|---|---|
| 1 | Configuration: add `PRAGMA busy_timeout`, configure connection pool. No mutex removed yet. Verify concurrency tests still pass. |
| 2 | Remove mutex on **read-only paths**: search, stats, list operations. These are the lowest risk and immediately unlock parallel reads. |
| 3 | Remove mutex on **write paths** and importers. Convert Class B sites to explicit transactions. Keep only Class C protections. |

## Tests

Add new tests under `pkg/blunderdb/storage/sqlite/` (and Postgres):

- [ ] `concurrent_reads_test.go` — 100 goroutines doing `Positions().List`
      concurrently while another goroutine inserts; no deadlock, no
      `SQLITE_BUSY`, no lost writes.
- [ ] `concurrent_writes_test.go` — 100 goroutines × 100 inserts to
      different Zobrist hashes; final row count = 10 000.
- [ ] `mixed_workload_test.go` — 10 readers + 5 writers + 1 long search,
      bounded duration, no errors.

These should also pass on Postgres trivially (no global mutex was ever
applied there).

## Gotchas

1. **`SetupDatabase` / `OpenDatabase` mutate `d.db`.** Today the GUI
   re-uses a single `Database` instance and swaps the underlying `*sql.DB`
   when the user opens a different file. Two options:
   - **Keep this pattern**: protect the swap with a tiny `sync.Mutex`
     dedicated to the `db` field, and lock only around `d.db = newDB`.
   - **Refactor**: create a new `Database` each time. Cleaner but requires
     adjusting the Wails binding lifecycle. Defer to a later cleanup.
   Recommendation: keep the pattern, fine-grained mutex on the swap.
2. **`importCancelled int32`** is an atomic flag, not a mutex. It stays
   for now; it goes away in [P8](08-streaming-imports-ctx.md).
3. **Stats queries scan large tables.** Without the mutex, a long stats
   query no longer blocks writes — but it may observe partially-committed
   data depending on isolation. Document expected isolation (`READ
   COMMITTED` semantics).
4. **`db_migration.CheckVersion`** runs at startup and mutates schema. Keep
   it serialized — either run once before exposing the `Storage` to
   callers, or guard with a `sync.Once`.
5. **Tests at the repo root that called `Database` directly** may have
   implicitly relied on serialisation (e.g. "do A then B and read the
   state"). Audit each test for hidden ordering assumptions.

## Performance expectation

After this phase, the SQLite backend should sustain ~1-2 k ops/s on a
laptop SSD (limited by SQLite's single-writer property). Postgres should
sustain ~10 k ops/s with `pool=50`. Confirmed empirically in
[P9](09-benchmarks.md).

## Verification

- [ ] `grep -rn '\.mu\.' pkg/blunderdb/` returns only documented
      Class-C callsites (≤ 5 expected, all in `Database` wrapper for
      lifecycle).
- [ ] All existing tests green.
- [ ] New concurrent tests green.
- [ ] Microbench (P9 scaffold) shows ≥ 5× improvement in write throughput
      on SQLite under contention.

## Risks

- Race conditions that were masked by the global mutex now surface. The
  contract test ([P2](02-storage-interface.md)) and the new concurrent
  tests should catch them; if not, `go test -race ./...` is mandatory in
  CI for this phase.
- Subtle behavioural changes in long-running stats queries (now reading
  uncommitted writes). Acceptable for blunderDB's read patterns, but
  document the isolation in `pkg/blunderdb/storage/doc.go`.
