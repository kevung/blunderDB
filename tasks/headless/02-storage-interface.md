# P2 — `Storage` interface + SQLite implementation

**Goal.** Define a `Storage` Go interface (composed of per-family
sub-interfaces) covering every persistence operation, and refactor the
SQLite code under `pkg/blunderdb/storage/sqlite/` to satisfy it. The
`Database` wrapper (kept for Wails) delegates to a `Storage`. Behaviour is
preserved 100 %.

**Estimate.** 8-10 days. **Risk.** High (≈186 methods mapped). **PRs.** 6
(split by family).

**Prerequisites.** [P1](01b-pkg-library-refactor.md).

## Why an interface

- Enables a second backend ([P3](03-postgres-backend.md) PostgreSQL).
- Forces every method to take `context.Context` (precondition for [P8
  cancellation](08-streaming-imports-ctx.md)).
- Cleanly separates persistence from domain logic / helpers (Zobrist
  hashing, compression, populating denormalized columns).

## Target layout

```
pkg/blunderdb/storage/
  storage.go              ← composes the sub-interfaces; root Storage type
  tx.go                   ← Tx interface (BeginTx/Commit/Rollback)
  positions.go            ← PositionStore
  analyses.go             ← AnalysisStore
  matches.go              ← MatchStore (Match, Game, Move, MoveAnalysis)
  comments.go             ← CommentStore
  collections.go          ← CollectionStore
  tournaments.go          ← TournamentStore
  anki.go                 ← AnkiStore
  filters.go              ← FilterStore
  session.go              ← SessionStore  (scope-aware, see P4)
  search.go               ← SearchStore + SearchHistoryStore
  stats.go                ← StatsStore
  history.go              ← CommandHistoryStore
  metadata.go             ← MetadataStore (database_version, MET ID, …)
  errors.go               ← typed errors: ErrNotFound, ErrConflict, …
  contract_test.go        ← parametrised test suite (reused by Postgres)
  sqlite/
    sqlite.go             ← type Storage + Open(ctx, dsn, opts) + pragmas
    tx_sqlite.go
    positions_sqlite.go
    analyses_sqlite.go
    matches_sqlite.go
    comments_sqlite.go
    collections_sqlite.go
    tournaments_sqlite.go
    anki_sqlite.go
    filters_sqlite.go
    session_sqlite.go
    search_sqlite.go
    stats_sqlite.go
    history_sqlite.go
    metadata_sqlite.go
    schema_sqlite.go      ← CREATE TABLE / INDEX (current v2.7.0 DDL)
    migration_sqlite.go   ← migrations 1.0.0 → 2.7.0 (current logic)
    sqlite_test.go        ← calls contract_test.Run(t, openSQLiteMem)
```

## Interface design choices

```go
package storage

type Storage interface {
    Positions()         PositionStore
    Analyses()          AnalysisStore
    Matches()           MatchStore
    Comments()          CommentStore
    Collections()       CollectionStore
    Tournaments()       TournamentStore
    Anki()              AnkiStore
    Filters()           FilterStore
    Session()           SessionStore
    Search()            SearchStore
    Stats()             StatsStore
    History()           CommandHistoryStore
    Metadata()          MetadataStore

    BeginTx(ctx context.Context) (Tx, error)
    Close() error
    Version(ctx context.Context) (string, error)
    Migrate(ctx context.Context) error
}

type PositionStore interface {
    Save(ctx context.Context, scope string, p *domain.Position) (int64, error)
    Update(ctx context.Context, scope string, p *domain.Position) error
    Load(ctx context.Context, scope string, id int64) (*domain.Position, error)
    Exists(ctx context.Context, scope string, zobrist uint64) (int64, bool, error)
    Delete(ctx context.Context, scope string, id int64) error
    List(ctx context.Context, scope string, opts ListOpts) iter.Seq2[*domain.Position, error]
}
```

Notes:
- `scope` is empty `""` in this phase (gets meaning in
  [P4](04-session-scope.md) for session-like state, then in
  [P3](03-postgres-backend.md) as the per-tenant identifier).
- `List` returns a `range`-friendly iterator (`iter.Seq2` from Go 1.23+) to
  stream large result sets — used in [P6](06-serve-http.md) NDJSON
  responses.
- Every `Save/Update/Delete` returns `(int64, error)` or `error`. SQLite
  uses `LastInsertId` internally; PostgreSQL will use `RETURNING id`. The
  interface hides the difference.
- `Tx` exposes the same sub-stores via `WithTx(tx)` helpers, so any method
  can be transactional:
  ```go
  s.Positions().Save(ctx, scope, p)              // implicit autocommit
  tx, _ := s.BeginTx(ctx)
  defer tx.Rollback()
  storage.WithTx(tx).Positions().Save(ctx, scope, p)
  ...
  tx.Commit()
  ```

## Non-storage helpers (stay outside the interface)

These should **not** be on `Storage`. They are pure functions or
engine-side helpers:

- Zobrist hashing (`engine/zobrist.go`).
- zlib compression / decompression of `analysis.data`.
- Population of denormalized scalar columns on `position` /
  `analysis` (decision_type, dice_1/2, pip_diff, off_1/2,
  back_checkers_1/2, occupancy_1/2, point_mask_1/2, …). Refactor as
  pure functions accepting a `*Position` and returning the column values.
- Canonical hash computation (already in `canonical_hash_test.go` —
  belongs in `engine/`).

## Contract test

`pkg/blunderdb/storage/contract_test.go`:

```go
func RunContractTests(t *testing.T, factory func() Storage) {
    t.Run("Position/Save+Load", func(t *testing.T) { … })
    t.Run("Position/DedupByZobrist", func(t *testing.T) { … })
    t.Run("Position/UpdatePreservesId", func(t *testing.T) { … })
    t.Run("Analysis/SaveAndCompress", func(t *testing.T) { … })
    t.Run("Match/CreateGameMoveCascade", func(t *testing.T) { … })
    t.Run("Match/DeleteCascade", func(t *testing.T) { … })
    t.Run("Tournament/AddRemoveMatch", func(t *testing.T) { … })
    t.Run("Collection/MoveBetweenCollections", func(t *testing.T) { … })
    t.Run("Anki/ReviewUpdatesScheduling", func(t *testing.T) { … })
    t.Run("Filter/SaveAndList", func(t *testing.T) { … })
    t.Run("Session/SaveLoadEmpty", func(t *testing.T) { … })
    t.Run("Search/FilterByDecisionType", func(t *testing.T) { … })
    t.Run("Stats/AggregateCounts", func(t *testing.T) { … })
    t.Run("Tx/RollbackUndoes", func(t *testing.T) { … })
    t.Run("Tx/CommitPersists", func(t *testing.T) { … })
}
```

SQLite test runner:

```go
// pkg/blunderdb/storage/sqlite/sqlite_test.go
func TestContract_SQLite(t *testing.T) {
    storage.RunContractTests(t, func() storage.Storage {
        s, _ := sqlite.Open(context.Background(), ":memory:", nil)
        return s
    })
}
```

The same `RunContractTests` is reused for Postgres in P3.

## Mutex / concurrency

The current global `Database.mu` **disappears from the public interface**.
Each implementation is free to choose its concurrency strategy. SQLite
implementation keeps an internal mutex *only* if needed for cross-statement
invariants (audit during this phase). Detailed audit and aggressive removal
happen in [P5](05-remove-global-mutex.md); P2 should not regress
concurrency.

## Splitting into 6 PRs

| PR | Scope |
|---|---|
| 1 | `storage.go` + `tx.go` + sub-interface definitions (no impl). `storage/errors.go`. `contract_test.go` skeleton. |
| 2 | SQLite impl: positions + analyses + search basics. Wire `Database` wrapper to delegate to `Storage` for these families. |
| 3 | SQLite impl: matches + games + moves + move_analyses (cascade behaviour). Tournaments. |
| 4 | SQLite impl: collections + comments + anki. |
| 5 | SQLite impl: filters + session + search-history + command-history + metadata. |
| 6 | SQLite impl: stats + migrations + schema bootstrap. `Database` wrapper fully delegating; pull out the last direct `*sql.DB` usage from `Database`. |

## Gotchas specific to blunderDB

1. **`PositionExists` (`db_position.go:8`) is currently not used inside
   `SavePosition`** — that path does an `INSERT … ON CONFLICT DO NOTHING`
   keyed on Zobrist. Audit who calls `PositionExists` separately and what
   it expects.
2. **`analysis.data` is zlib-compressed**. Compress/decompress lives in
   helpers, not on the interface. The interface returns
   `*domain.Analysis` already decompressed.
3. **`migrationProgress`** callback (an option of the `Database` wrapper)
   threads through to `Storage.Migrate(ctx)`. Add an option to
   `storage.Open(ctx, dsn, opts)`:
   ```go
   opts := &storage.Options{
       MigrationProgress: func(phase string, done, total int) { … },
   }
   ```
4. **`importCancelled int32`** (atomic global on `Database`) must not be
   part of the `Storage` interface. In this phase, keep the existing global
   flag as-is to avoid scope creep; it goes away in
   [P8](08-streaming-imports-ctx.md) when `ctx.Done()` replaces it.
5. **Auto-incrementing IDs.** SQLite `INTEGER PRIMARY KEY AUTOINCREMENT`
   gives `LastInsertId()` via `database/sql`. Postgres needs `RETURNING
   id`. The interface signature `Save(...) (int64, error)` already abstracts
   this; just make sure the SQLite impl uses `result.LastInsertId()`.
6. **Wails binding stability.** Throughout this refactor, the `Database`
   wrapper's public method signatures must not change. Add internal
   delegation, but keep the surface identical.

## Verification

- [ ] `go test ./pkg/blunderdb/storage/...` green (contract test on SQLite).
- [ ] `go test ./...` green (existing tests via `Database` wrapper).
- [ ] `wails build` + smoke test: GUI still loads, opens a DB, saves a
      position, runs a search.
- [ ] `grep -rn 'd\.db\.' pkg/blunderdb/database/` ≤ 1 callsite (only the
      delegation hook). All other accesses go via `s.<family>().<method>()`.

## Risks

- Subtle behaviour drift: e.g. an `INSERT OR REPLACE` becoming `INSERT …
  ON CONFLICT … DO UPDATE` is fine in SQLite but must produce the same
  observed row state. Contract test must cover.
- Cross-family transactions: `DeleteMatch` cascades manually through
  games/moves/analyses. After the split into per-family stores, the cascade
  must still happen inside a single `Tx`. Add a `MatchStore.DeleteCascade(tx,
  scope, matchID)` helper rather than letting callers stitch the cascade
  manually.
