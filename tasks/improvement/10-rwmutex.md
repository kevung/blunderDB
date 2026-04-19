# 10 — Switch `sync.Mutex` to `sync.RWMutex`

**Goal:** Allow concurrent read operations on the database by using `RWMutex` instead of exclusive `Mutex`. Improves GUI responsiveness during imports.

**Depends on:** 06 (db.go split — mutex is used across all db_*.go files).

**Impact:** Medium — reads currently block reads unnecessarily.

## Context

- `Database.mu` is a `sync.Mutex` (db.go L31 pre-split)
- **110 `d.mu.Lock()` calls** across all Database methods
- Every public method — reads AND writes — acquires the same exclusive lock
- SQLite WAL mode is already enabled, which supports concurrent readers at the DB level
- The mutex only needs to prevent concurrent Go-level transaction conflicts

### Read-only methods (~60, use `RLock`)

Methods that only `SELECT` and never modify the database:

`GetAllMatches`, `GetMatchByID`, `GetGamesByMatch`, `GetMovesByGame`, `GetMatchMovePositions`, `GetLastVisitedMatch`, `GetDatabaseStats`, `GetAllCollections`, `GetCollectionByID`, `GetCollectionPositions`, `GetPositionCollections`, `GetAllTournaments`, `GetTournamentMatches`, `GetMatchTournament`, `GetAllAnkiDecks`, `GetAnkiDeckPositions`, `GetAnkiDeckStats`, `GetNextAnkiCard`, `LoadPosition`, `LoadAnalysis`, `LoadAllPositions`, `LoadPositionsByFilters`, `loadPositionsByFiltersCore`, `LoadComment`, `GetCommentsByPosition`, `GetAllComments`, `SearchComments`, `LoadCommandHistory`, `LoadSearchHistory`, `LoadSessionState`, `PositionExists`, `CheckMatchExists`, `CheckVersion`, `CheckDatabaseVersion`, `LoadFilters`, `GetPositionIndexMap`, `AnalyzeImportDatabase`, `loadPositionByIDUnlocked`

### Write methods (~50, keep `Lock`)

Methods that `INSERT`, `UPDATE`, `DELETE`, or run transactions:

`SetupDatabase`, `OpenDatabase`, `SavePosition`, `UpdatePosition`, `DeletePosition`, `SaveAnalysis`, `DeleteAnalysis`, `SaveComment`, `AddComment`, `UpdateCommentEntry`, `DeleteCommentEntry`, `ImportXGMatch`, `ImportGnuBGMatch`, `ImportBGFMatch`, `ExportDatabase`, `CommitImportDatabase`, `CreateCollection`, `UpdateCollection`, `DeleteCollection`, `AddPositionToCollection`, `RemovePositionFromCollection`, `CreateTournament`, `UpdateTournament`, `DeleteTournament`, `AddMatchToTournament`, `DeleteMatch`, `CreateAnkiDeck`, `UpdateAnkiDeck`, `DeleteAnkiDeck`, `SyncAnkiDeck`, `ReviewAnkiCard`, `ResetAnkiDeck`, `SaveCommand`, `SaveSearchHistory`, `DeleteSearchHistoryEntry`, `SaveSessionState`, `ClearSessionState`, `SaveLastVisitedPosition`, `SaveFilter`, all `migrate_*` functions, `ensureAllTablesExist`

## Files touched

- **Edit:** Struct definition file (post-split `db.go` or wherever `Database` struct is defined)
- **Edit:** All `db_*.go` files — change `d.mu.Lock()` to `d.mu.RLock()` for read-only methods

## Tasks

### 1. Change mutex type

- [ ] In the `Database` struct definition:
  ```go
  // Before:
  mu sync.Mutex
  // After:
  mu sync.RWMutex
  ```
- [ ] This is a one-line change; `RWMutex` has `Lock()`/`Unlock()` (same as `Mutex`) plus `RLock()`/`RUnlock()`

### 2. Classify methods

- [ ] For each of the ~110 `d.mu.Lock()` calls, determine if the method is read-only or read-write
- [ ] A method is read-only if it only executes `db.Query`, `db.QueryRow`, `db.Prepare` + `stmt.Query`
- [ ] A method is read-write if it executes `db.Exec`, `db.Begin`, or calls a write helper within a transaction

### 3. Replace lock calls in read-only methods

- [ ] For each read-only method, replace:
  ```go
  // Before:
  d.mu.Lock()
  defer d.mu.Unlock()
  // After:
  d.mu.RLock()
  defer d.mu.RUnlock()
  ```
- [ ] Process one `db_*.go` file at a time

### 4. Verify with race detector

- [ ] Run `go test -race ./...` — the race detector will catch any misclassification
- [ ] If a "read-only" method actually writes (missed during classification), the race detector will flag it
- [ ] Fix any races found

### 5. Verify no deadlocks

- [ ] Ensure no read-only method calls a write method while holding `RLock` (this would deadlock)
- [ ] Review call chains: `LoadPositionsByFilters` → `loadPositionsByFiltersCore` → any write? (should be no)
- [ ] `AnalyzeImportDatabase` reads extensively — confirm it doesn't write

## Acceptance criteria

- [ ] `Database.mu` is `sync.RWMutex`
- [ ] ~60 read-only methods use `RLock()`/`RUnlock()`
- [ ] ~50 write methods continue using `Lock()`/`Unlock()`
- [ ] `go test -race ./...` passes with zero races
- [ ] No deadlocks in normal operation (concurrent reads don't block each other)

## Rollback

`git revert` — change `sync.RWMutex` back to `sync.Mutex` and `RLock/RUnlock` back to `Lock/Unlock`. Fully mechanical.
