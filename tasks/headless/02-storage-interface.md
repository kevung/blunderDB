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

## Arbitrated decisions (2026-05-21, after PR1)

Open design questions between this sheet and what PR1 actually shipped were
settled before starting PR2-6:

- **D1 — `PositionStore.Save` deduplicates by Zobrist hash.** `Save` becomes
  idempotent (`INSERT … ON CONFLICT(zobrist_hash) DO NOTHING`, then returns the
  row id). Matches the `positions.go` doc comment and the `DedupByZobrist`
  contract case; fixes the latent failure where a plain `INSERT` violates the
  unique `idx_position_zobrist`. The import path must be audited so it does not
  rely on the constraint error.
- **D2 — uniform transaction model.** `MatchStore.DeleteCascade` drops its
  explicit `tx Tx` parameter (`DeleteCascade(ctx, scope, id)`); `Tx` already
  embeds `Stores`. **Applied in PR3.** No cross-family operation needs a
  caller-passed `Tx`. Each backend's `DeleteCascade` opens its own
  transaction; when the store is reached through a `Tx` it joins that
  transaction instead (SQLite runs the statements directly on the caller's
  `*sql.Tx`; PostgreSQL opens a savepoint).
- **D3 — schema bootstrap moves forward to PR2** (was PR6), so PR2 is testable
  against a real schema on an empty `:memory:` DB. The 1.0→2.7 migration chain
  stays in PR6.
- **D4 — `SearchStore.Find` is ported whole in PR2.** `LoadPositionsByFiltersCore`
  is one tightly-coupled algorithm; "search basics" means port it entire, not
  carve a subset.
- **D5 — shared `*sql.DB`.** `sqlite` exposes `Open(ctx,dsn,opts)` (owns the
  handle) and `New(db)` (borrows it; `Close` is a no-op). The `Database` wrapper
  rebuilds its `Storage` at the end of `SetupDatabase`/`OpenDatabase`.
- **D6 — `Storage.Version()` delegates to `MetadataStore.Version()`.** Both kept.
- **D7 — DTO policy.** Wrapper keeps its `database.*` return types (Wails-bound);
  it converts `storage.X ↔ database.X` at the delegation boundary. No promotion
  to `domain` in P2.
- **D8 — shared helpers extracted.** `populatePositionColumns`, the compact-board
  codec and the `*Database`-receiver search helpers move out of `database` (it
  imports `storage/sqlite`, so `storage/sqlite` cannot import it back).
- **D9 — shared pragmas.** Pragma list (incl. `foreign_keys=ON`) moves to a
  shared `sqlite` function used by both `Open` and the wrapper.
- **D10 — `LoadPositionsByFiltersCore` stays a private wrapper method** until
  PR5; only the Wails-bound `LoadPositionsByFilters` delegates to `Find` in PR2.

## Splitting into 6 PRs

| PR | Scope |
|---|---|
| 1 | `storage.go` + `tx.go` + sub-interface definitions (no impl). `storage/errors.go`. `storagetest/contract.go` skeleton. **Done (`c8bcfc1e`).** |
| 2 | Shared-helper extraction (D8). SQLite schema bootstrap (D3). SQLite impl: positions + analyses + full search (D4). `sqlite.Open`/`New` (D5), shared pragmas (D9). Wire `Database` to delegate the positions/analyses/search families. Contract tests for positions/analyses/search/tx. **Done.** |
| 3 | **Done.** SQLite impl: matches + games + moves (`matchStore`) and tournaments (`tournamentStore`). `DeleteCascade` amendment (D2) applied — the interface drops the `tx Tx` parameter and each backend runs its own cascade transaction (`withTx` for SQLite, savepoint-capable `inTx` for PostgreSQL; the PostgreSQL P3 PR3 store was updated to match). Shared contract `Match/*` cases enabled (green on both backends); the `Tournament/AddRemoveMatch` case stays pending until P3 PR4 lands the PostgreSQL tournament store. SQLite-specific tests cover the methods the contract omits (`SwapPlayers`, `MergePlayers`, `LastVisited`, tournament membership). Move analyses cascade off the match delete via `ON DELETE CASCADE` (no dedicated store). The `Database` wrapper still owns the match/tournament SQL — wrapper delegation deferred. |
| 4 | **Done.** SQLite impl: collections (`collectionStore`), comments (`commentStore`) and anki/FSRS (`ankiStore`). Shared contract `Collection/MoveBetweenCollections` enabled (green on both backends — PostgreSQL collections landed in P3 PR4); the `Anki/ReviewUpdatesScheduling` case stays pending until P3 PR5 lands the PostgreSQL anki store. SQLite-specific tests cover what the contract omits (collection CRUD / membership / index-map, comment CRUD / concatenation / search, anki deck CRUD / sync / review lifecycle). `commentStore` drops the legacy `commentTableHasTimestamps` probe — the v2.7.0 bootstrap always provides the `created_at` / `modified_at` columns. The `Database` wrapper still owns the collection/comment/anki SQL — wrapper delegation and the D7 DTO conversion helpers deferred. |
| 5 | **Done.** SQLite impl: filters (`filterStore`), session (`sessionStore`), search-history (`searchHistoryStore`), command-history (`commandHistoryStore`) and metadata (`metadataStore`). D6 applied — `sqlite.Storage.Version` delegates to `MetadataStore.Version`. D10 applied — `LoadPositionsByFiltersCore` now delegates to `SearchStore.Find` + `AnalysisStore.Load`; the ~445-line duplicated SQL search implementation and its 7 private helpers were deleted from `database/db_search.go` (the equivalence tests in `search_rewrite_test.go` keep their own legacy reference copy). Migration-era `database_version` version guards dropped — the v2.7.0 bootstrap always provides the tables. List ordering gained `id` tiebreakers (`search_history`, `command_history`) for determinism under same-millisecond timestamps. Contract cases `Filter/SaveAndList` and `Session/SaveLoadEmpty` stay pending until P3 PR5 lands the PostgreSQL filter/session stores; SQLite-specific tests cover all five families. The `Database` wrapper still owns the filter/session/history/metadata SQL — wrapper delegation deferred to PR6. |
| 6 | SQLite impl: stats + migrations + schema-version finalisation. `Database` wrapper fully delegating; pull out the last direct `*sql.DB` usage from `Database`. |

## Gotchas specific to blunderDB

1. **`PositionExists` (`db_position.go:10`) is currently not used inside
   `SavePosition`.** `SavePosition` does a plain `INSERT` (no `ON CONFLICT`),
   so a duplicate Zobrist hash would violate the unique `idx_position_zobrist`.
   Per D1, `PositionStore.Save` introduces the `ON CONFLICT DO NOTHING` dedup.
   `PositionExists` itself does a full-table JSON comparison (not a Zobrist
   lookup) — keep it on raw SQL in PR2; audit its callers separately.
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
