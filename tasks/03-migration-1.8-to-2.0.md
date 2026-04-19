# 03 — Migration 1.8.0 → 2.0.0

**Goal:** Existing 1.8.0 databases open cleanly under the 2.0.0 binary: ALTER TABLE + backfill + indexes, inside one transaction, with progress events for the GUI.

**Depends on:** 02 (fresh-DB schema is correct; populator helpers exist).

## Files touched

- `db.go` — extend the migration chain near `db.go:430+`, and adjust `CheckVersion` / `OpenDatabase`.
- `migration_test.go` — add 1.8.0 → 2.0.0 coverage.
- `app.go` — expose a Wails-bound `OnMigrationProgress` event channel if not already there (reuse the import-progress pattern).

## 1. Where the step lives

- [ ] Find the v1.7.0 → v1.8.0 block in `db.go` (follow the pattern of the preceding versions, each queries `sqlite_master` then CREATE/ALTER).
- [ ] Add a new `migrate_1_8_0_to_2_0_0(tx *sql.Tx) error` function, called from `OpenDatabase` after the 1.8.0 step.

## 2. Migration body

### 2.1 Guard

- [ ] Read the stored version from `metadata.value WHERE key='database_version'`. If already `"2.0.0"`, return immediately.
- [ ] If the `position` table is empty, skip the backfill but still run the ALTERs + index creation (fresh-but-opened-from-older-binary case).

### 2.2 ALTER TABLE (nullable columns)

Wrap in one transaction with `BEGIN IMMEDIATE`:

```sql
ALTER TABLE position ADD COLUMN zobrist_hash    INTEGER;
ALTER TABLE position ADD COLUMN decision_type   INTEGER;
ALTER TABLE position ADD COLUMN player_on_roll  INTEGER;
ALTER TABLE position ADD COLUMN dice_1          INTEGER;
ALTER TABLE position ADD COLUMN dice_2          INTEGER;
ALTER TABLE position ADD COLUMN cube_value      INTEGER;
ALTER TABLE position ADD COLUMN cube_owner      INTEGER;
ALTER TABLE position ADD COLUMN score_1         INTEGER;
ALTER TABLE position ADD COLUMN score_2         INTEGER;
ALTER TABLE position ADD COLUMN match_length    INTEGER;
ALTER TABLE position ADD COLUMN has_jacoby      INTEGER;
ALTER TABLE position ADD COLUMN has_beaver      INTEGER;
ALTER TABLE position ADD COLUMN pip_1           INTEGER;
ALTER TABLE position ADD COLUMN pip_2           INTEGER;
ALTER TABLE position ADD COLUMN pip_diff        INTEGER;
ALTER TABLE position ADD COLUMN off_1           INTEGER;
ALTER TABLE position ADD COLUMN off_2           INTEGER;
ALTER TABLE position ADD COLUMN back_checkers_1 INTEGER;
ALTER TABLE position ADD COLUMN back_checkers_2 INTEGER;
ALTER TABLE position ADD COLUMN no_contact      INTEGER;
ALTER TABLE position ADD COLUMN occupancy_1     INTEGER;
ALTER TABLE position ADD COLUMN occupancy_2     INTEGER;
ALTER TABLE position ADD COLUMN point_mask_1    INTEGER;
ALTER TABLE position ADD COLUMN point_mask_2    INTEGER;

ALTER TABLE analysis ADD COLUMN best_cube_action        TEXT;
ALTER TABLE analysis ADD COLUMN cube_error              REAL;
ALTER TABLE analysis ADD COLUMN best_move_equity_error  REAL;
ALTER TABLE analysis ADD COLUMN player1_win_rate        REAL;
ALTER TABLE analysis ADD COLUMN player1_gammon_rate     REAL;
ALTER TABLE analysis ADD COLUMN player1_backgammon_rate REAL;
ALTER TABLE analysis ADD COLUMN player2_win_rate        REAL;
ALTER TABLE analysis ADD COLUMN player2_gammon_rate     REAL;
ALTER TABLE analysis ADD COLUMN player2_backgammon_rate REAL;
```

- [ ] Each ALTER is idempotent: if it fails with "duplicate column name", swallow the error (mirrors the existing ALTER pattern at db.go:1037–1048).

### 2.3 Backfill `position`

- [ ] `SELECT COUNT(*) FROM position` → total for progress.
- [ ] `SELECT id, state FROM position ORDER BY id` — process in batches of 1000.
- [ ] For each row: `json.Unmarshal` → `populatePositionColumns` → `UPDATE position SET ... WHERE id=?`.
- [ ] Use a single prepared `UPDATE` statement reused across rows.
- [ ] Emit `EventsEmit(a.ctx, "migration:progress", {phase: "position", done, total})` every ~200 rows.
- [ ] Cancellation: check `atomic.LoadInt32(&d.importCancelled)` every batch. On cancel, ROLLBACK and return a sentinel error.

### 2.4 Backfill `analysis`

- [ ] `SELECT id, position_id, data FROM analysis ORDER BY id`, same batch pattern.
- [ ] For each row: parse `data` JSON → `populateAnalysisColumns(…)` → `UPDATE analysis SET ... WHERE id=?`.
- [ ] Some analyses may miss the played-move context — tolerate `NULL` in `best_move_equity_error` / `cube_error`.

### 2.5 Dedup check (before unique index)

- [ ] `SELECT zobrist_hash, COUNT(*) c FROM position GROUP BY zobrist_hash HAVING c > 1`.
- [ ] If any duplicates: surface them to the user via a migration-failure dialog ("N duplicate positions detected"). Offer a best-effort merge:
  - Keep the lowest `position.id`.
  - Re-point any `move.position_id`, `collection_position.position_id`, `anki_card.position_id` from the discarded ids to the kept id.
  - Delete the now-orphan `position` / `analysis` rows.
- [ ] Log a summary: "Merged N duplicate positions during 2.0.0 migration".

### 2.6 Create indexes + ANALYZE

- [ ] Run every `CREATE INDEX IF NOT EXISTS` from phase 02's index list.
- [ ] `ANALYZE;` at the end so the query planner picks up distribution stats before the first search runs.

### 2.7 Bump version

- [ ] `UPDATE metadata SET value='2.0.0' WHERE key='database_version'`.
- [ ] `tx.Commit()`.

## 3. Wails progress UX

- [ ] In `OpenDatabase` (db.go:399), before running migrations, emit `"migration:start"` with `{fromVersion, toVersion}`.
- [ ] A small Svelte modal (phase 06) subscribes to `migration:progress` and `migration:done`. **Do not build the modal in this phase** — only emit the events; existing session UI keeps working since Wails tolerates unhandled events.

## 4. Tests (`migration_test.go`)

- [ ] Fixture: commit a tiny `testdata/v1.8.0_snapshot.db` (hand-crafted: 3 positions with analyses, 1 match, 2 games, 5 moves, 1 collection, 1 tournament). ~20 KB.
- [ ] `TestMigrate_1_8_0_to_2_0_0`:
  - Copy fixture to temp path, open with v2.0.0 binary path.
  - Assert `database_version = '2.0.0'`.
  - Assert every new column on every row is non-NULL (except legitimately-nullable `cube_value`, `cube_owner`, `best_move_equity_error`, etc.).
  - Recompute expected values (via `populatePositionColumns` on the decoded JSON) and compare to stored columns — drift = failure.
  - Assert indexes exist via `SELECT name FROM sqlite_master WHERE type='index'`.
- [ ] `TestMigrate_1_8_0_Duplicates`: craft a fixture with two positions whose canonicalized JSON collides. Assert the dedup step merges them and remaps `move.position_id`.
- [ ] `TestMigrate_Idempotent`: run migration twice in a row — second run is a no-op.

## Acceptance criteria

- [ ] `go test -run TestMigrate ./...` green.
- [ ] `./blunderdb list --db <1.8.0 copy> --type stats` auto-migrates and returns identical row counts to pre-migration stats.
- [ ] `./blunderdb search --db <migrated DB> --decision cube` returns the same IDs as the pre-migration binary would have.
- [ ] Manual GUI: open a real 1.8.0 DB (kept under `~/bdb-prod-copy.db`, never committed). Migration progress bar visible, DB opens, every filter still returns results.

## Risks

- **Long backfill on large DBs.** A 100k-position DB with analyses could take a minute. Progress events are the mitigation; also advise users in the release notes to back up before upgrade.
- **Partial migration.** If the process is killed mid-backfill and the transaction aborts, columns exist but are mostly NULL. On next open, the version check still says 1.8.0 (we haven't committed the version bump), so the migration runs again from scratch. The ALTERs are idempotent (swallow "duplicate column"). **Verify by killing `./blunderdb` during a migration test run.**
- **Duplicate-position merge bugs.** Re-pointing foreign keys across four tables is where migrations go wrong. Cover with the duplicates test; in production log the remapping table to `~/.config/blunderDB/migrations/2.0.0-merges.log` so a user can audit.
- **FK cascade on DELETE during dedup.** `move.position_id` is FK `SET NULL` (db.go:942) — safe. `collection_position.position_id` is `CASCADE` (db.go:274) — deleting the wrong `position` id would silently drop collection entries. Re-point BEFORE deleting.
