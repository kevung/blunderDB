# 04 — Import rewrite

**Goal:** Remove the in-RAM position warmup, add prepared statements, add cancellation in the hot loop, and checkpoint WAL between files.

**Depends on:** 03 (schema + migration in place; the unique `zobrist_hash` index is the new dedup oracle).

## Files touched

- `db.go` — every `ImportXXXMatch` entry point and the shared per-move helpers.
- `cli.go` — `importBatch`.
- No schema change.

## 1. Remove the preload warmup

- [ ] In `ImportXGMatch` (db.go:5652), delete the block that does `SELECT id, state FROM position` and builds `positionCache` / `positionCacheByJSON` (db.go:5805–5837).
- [ ] Same treatment in `importGnuBGMatchInternal` (db.go:8256) and `ImportBGFMatch` (db.go:9936) — track down the analogous warmup loops.
- [ ] Remove now-unused helper functions that only served the warmup (`loadExistingPositionCache`, etc.). `go build` will flag them.

## 2. Prepared statements per transaction

Factor a single `type importStmts struct { insertPosition, selectPositionID, insertAnalysis, insertMove, insertMoveAnalysis *sql.Stmt; close func() }`. One constructor `(tx *sql.Tx) (*importStmts, error)`.

- [ ] `insertPosition`:
  ```sql
  INSERT OR IGNORE INTO position (
    zobrist_hash, decision_type, player_on_roll,
    dice_1, dice_2, cube_value, cube_owner,
    score_1, score_2, match_length, has_jacoby, has_beaver,
    pip_1, pip_2, pip_diff,
    off_1, off_2, back_checkers_1, back_checkers_2,
    no_contact, occupancy_1, occupancy_2, point_mask_1, point_mask_2,
    state
  ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
  ```
- [ ] `selectPositionID`: `SELECT id FROM position WHERE zobrist_hash = ?`.
- [ ] `insertAnalysis`: full column list including the new ones, keep `data` JSON at the end.
- [ ] `insertMove` / `insertMoveAnalysis`: straight ports of the existing INSERTs.
- [ ] `close` walks each `Stmt` and closes; call via `defer stmts.close()` at the top of every import.

### Per-transaction cache

- [ ] Small `map[uint64]int64` (`zobristHash → positionID`) lives on `importStmts`. Before any SQL, check the map. After `INSERT OR IGNORE` + `SELECT id`, stash the result. Avoids index probes for positions that repeat inside a single match (common in imports with many analyses of the same bearoff position).

## 3. Replace `savePositionInTxWithCache`

The existing function (db.go around 6688) normalizes JSON, does the map lookup, INSERTs, UPDATEs the ID back, and also writes collection links. Refactor into two stages:

- [ ] `(s *importStmts) saveOrReusePosition(tx *sql.Tx, p *Position) (int64, bool, error)` returns `(id, wasNew, err)`.
  - Normalize `p` (mirror if `PlayerOnRoll == 1`, same rule as the Zobrist step).
  - Compute all column values via `populatePositionColumns`.
  - `stmts.insertPosition.Exec(...)` — `INSERT OR IGNORE`.
  - `stmts.selectPositionID.QueryRow(zobristHash)` to get the id (works both for new and existing rows).
  - **Structural verification on dedup hit**: when `wasNew == false`, fetch the existing row's `state` JSON, decode it, and compare the board/cube/score fields to the incoming position. If they differ, it's a genuine Zobrist collision — log a warning and fall back to inserting with a disambiguated key (or skip with an error). This guards against the ~2⁻⁴⁴ collision risk identified in the plan.
  - Update cache, return.
- [ ] Callers in `importXGGamesAndMoves` / GnuBG / BGF analogues switch to the new helper.
- [ ] The old `state` JSON column is still written (phase 02 requirement). Keep it — the detail panel and any code not yet migrated still reads it.

## 4. `saveAnalysisInTx` update

- [ ] Use `populateAnalysisColumns` to fill the new columns alongside the JSON `data` blob.
- [ ] Guard: if an analysis row already exists for the `position_id`, the current code either UPDATEs or skips. Preserve that semantic; just extend the column list.

## 5. Cancellation responsiveness

- [ ] In the move loop (`importXGGamesAndMoves` db.go:5952), at the top of each iteration:
  ```go
  if atomic.LoadInt32(&d.importCancelled) != 0 {
      return errImportCancelled
  }
  ```
- [ ] Mirror in `importGnuBG` and `importBGF` move loops.
- [ ] Make sure the outer `defer tx.Rollback()` fires on cancel so no partial writes persist.

## 6. CLI batch checkpoints

- [ ] In `cli.go:importBatch` (around line 1334), after each file's import completes successfully:
  ```go
  cli.db.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
  ```
- [ ] On batch completion, run `ANALYZE;` once so search stats include the new rows. (Cheap relative to import.)

## 7. Drop per-position Wails progress flooding (if any)

- [ ] Audit `db.go` for `EventsEmit` inside the move loop. If any event fires per-move, batch it (emit every N moves) so the Svelte side isn't swamped.

## 8. Regression tests

- [ ] Pick 2–3 match files from `testdata/tournois/`. For each:
  - Import into a fresh in-memory DB with **old** code (run `git stash` + build, capture SQLite dump of `position` + `analysis` tables).
  - Import with **new** code.
  - Diff: same row count, same `state` JSON, same `data` JSON, same `move.position_id` linkages.
- [ ] Add a `TestImportIdempotence` that imports the same file twice; second import must result in zero new `position` rows (every insert hits the unique index).
- [ ] Add a `TestImportCancellation` that sets `importCancelled=1` mid-import via a goroutine; assert the DB is unchanged after the rollback.

## Acceptance criteria

- [ ] `go test ./...` and `go test ./tests/...` green.
- [ ] `BenchmarkImport_TournoisCold` from sheet 00: ≥3× faster than baseline.
- [ ] `BenchmarkImport_TournoisIncremental`: dramatic improvement (the whole preload is gone — expect 10×+ for large DBs).
- [ ] CLI `time find … -exec ./blunderdb import` on `testdata/tournois/` is visibly faster end-to-end.

## Risks

- **Double-count analyses.** If a position is reused across imports, the current code may write a second `analysis` row (the old cache distinguished "new position" from "reused position" when deciding whether to save analysis). Preserve this — the `saveOrReusePosition` return includes `wasNew`; the caller uses that to decide whether to `insertAnalysis`.
- **Transaction locking with WAL.** Multiple concurrent readers are fine under WAL; writers still serialize. The `Database.mu` in-process mutex already handles this. Don't remove it.
- **`INSERT OR IGNORE` + `SELECT` race.** Inside a single transaction there's no race. Between transactions, the unique index guarantees only one winner; losers pick up the id via `SELECT`. No application-level locking needed.
- **Prepared-statement caching across imports.** `importStmts` is per-transaction. Don't try to cache `*sql.Stmt` on `Database` — `modernc.org/sqlite` ties prepared statements to a connection, and the `*sql.DB` pool can move transactions to different connections. Per-tx is correct.
