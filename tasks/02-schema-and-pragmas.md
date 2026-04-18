# 02 — Schema & PRAGMAs (fresh DBs)

**Goal:** Every **freshly created** database has the v2.0.0 schema, the new indexes, and the tuning PRAGMAs applied. Existing databases are addressed in phase 03 — do not touch migration logic here.

**Depends on:** 01 (populators must exist even if still unused).

## Files touched

- `model.go` — bump `DatabaseVersion`.
- `db.go` — `SetupDatabase` schema block, `ensureAllTablesExist`, `OpenDatabase`, PRAGMA statements.

## 1. Bump version

- [ ] `model.go:36` → `DatabaseVersion = "2.0.0"`.
- [ ] Grep for any hard-coded `"1.8.0"` in db.go / tests; decide per-occurrence whether it's a migration anchor (keep) or a live version check (update). Migration-chain strings must NOT change.

## 2. New `position` table definition

Replace the existing `CREATE TABLE position` at `db.go:73–77` with the full v2.0.0 shape. Columns ordered to minimize SQLite row padding: fixed-width first, then text/blob.

```sql
CREATE TABLE IF NOT EXISTS position (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    zobrist_hash      INTEGER NOT NULL,
    decision_type     INTEGER NOT NULL,
    player_on_roll    INTEGER NOT NULL,
    dice_1            INTEGER NOT NULL,
    dice_2            INTEGER NOT NULL,
    cube_value        INTEGER,
    cube_owner        INTEGER,
    score_1           INTEGER NOT NULL,
    score_2           INTEGER NOT NULL,
    match_length      INTEGER NOT NULL,
    has_jacoby        INTEGER NOT NULL,
    has_beaver        INTEGER NOT NULL,
    pip_1             INTEGER NOT NULL,
    pip_2             INTEGER NOT NULL,
    pip_diff          INTEGER NOT NULL,
    off_1             INTEGER NOT NULL,
    off_2             INTEGER NOT NULL,
    back_checkers_1   INTEGER NOT NULL,
    back_checkers_2   INTEGER NOT NULL,
    no_contact        INTEGER NOT NULL,
    occupancy_1       INTEGER NOT NULL,
    occupancy_2       INTEGER NOT NULL,
    point_mask_1      INTEGER NOT NULL,
    point_mask_2      INTEGER NOT NULL,
    state             TEXT    NOT NULL
);
```

- [ ] Update `SavePosition` (db.go:1200) to populate every new column. Call `populatePositionColumns` from phase 01.
- [ ] Do the same in every import-side "save position" helper (`savePositionInTxWithCache`, XGP/BGF single-position paths). **Phase 04 will further refactor these** — for now, minimum correctness: new columns populated on every insert.

## 3. New `analysis` columns

The existing `analysis` table (db.go:84–90) already has `data JSON`. Append the new columns.

```sql
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

- [ ] In `SetupDatabase` use a single `CREATE TABLE` with all columns (no ALTERs needed for fresh DBs). The same ALTER list moves to phase 03 for existing DBs.
- [ ] Update `SaveAnalysis` (db.go:1265) to populate every new column via `populateAnalysisColumns`. Keep writing `data` JSON unchanged.

## 4. Indexes

Add after table creation in `SetupDatabase`, also in `ensureAllTablesExist` guarded by `CREATE INDEX IF NOT EXISTS`:

```sql
CREATE UNIQUE INDEX IF NOT EXISTS idx_position_zobrist       ON position(zobrist_hash);
CREATE        INDEX IF NOT EXISTS idx_position_decision_pip  ON position(decision_type, pip_diff);
CREATE        INDEX IF NOT EXISTS idx_position_decision_dice ON position(decision_type, dice_1, dice_2);
CREATE        INDEX IF NOT EXISTS idx_position_off           ON position(off_1, off_2);
CREATE        INDEX IF NOT EXISTS idx_position_score         ON position(match_length, score_1, score_2);
CREATE        INDEX IF NOT EXISTS idx_analysis_position      ON analysis(position_id);
CREATE        INDEX IF NOT EXISTS idx_analysis_win1          ON analysis(player1_win_rate);
CREATE        INDEX IF NOT EXISTS idx_analysis_cube_error    ON analysis(cube_error);
CREATE        INDEX IF NOT EXISTS idx_analysis_move_error    ON analysis(best_move_equity_error);
CREATE UNIQUE INDEX IF NOT EXISTS idx_match_canonical        ON match(canonical_hash);
CREATE        INDEX IF NOT EXISTS idx_move_position          ON move(position_id);
CREATE        INDEX IF NOT EXISTS idx_move_game              ON move(game_id);
```

- [ ] Note: `idx_match_canonical` replaces the existing non-unique `idx_match_hash` — keep the old one for back-compat (nothing breaks) and add the new unique one. Phase 03 is where we consider retiring duplicates.

## 5. PRAGMAs

Extract a `func (d *Database) applyPragmas() error` and call from both `SetupDatabase` (db.go:36) and `OpenDatabase` (db.go:399).

```sql
PRAGMA journal_mode = WAL;
PRAGMA synchronous  = NORMAL;
PRAGMA cache_size   = -65536;
PRAGMA temp_store   = MEMORY;
PRAGMA mmap_size    = 268435456;
PRAGMA foreign_keys = ON;
```

- [ ] In-memory DBs (`:memory:`) do not benefit from WAL — skip `journal_mode=WAL` when the path is `:memory:`; everything else still applies.
- [ ] Log the PRAGMA echoes at startup (the current code already prints DB setup errors; just don't swallow these).

## 6. Verification

- [ ] Fresh-DB smoke test: `go test -run TestSetupDatabase ./...` — add a new test if none exists that inspects `PRAGMA schema_version` and `sqlite_master` to confirm all expected tables/indexes/columns.
- [ ] Manual: `./blunderdb create --db /tmp/fresh.db` (or equivalent) then `sqlite3 /tmp/fresh.db '.schema position'` — should match the block above exactly.
- [ ] `PRAGMA journal_mode;` returns `wal` on a disk DB.

## Acceptance criteria

- [ ] `go test ./...` green.
- [ ] Fresh DB from `SetupDatabase` has every v2.0.0 column, every new index, and the PRAGMAs applied.
- [ ] Importing a single match file into a fresh DB succeeds (new columns populated via `SavePosition`/`SaveAnalysis`). Search still works — it's still the old slow path (rewrite is phase 05) but returns correct results.
- [ ] Re-running the baseline benchmarks on fresh-DB imports shows they have not **regressed** (extra columns cost a few %; if >10%, investigate).

## Risks

- **`NOT NULL` on backfill-able columns.** Fresh DBs are fine because `SavePosition` always sets them. The migration in phase 03 must add the columns as **nullable**, backfill, then (optionally) tighten — or just leave nullable forever, since the app only reads them. Decide now: *leave them nullable in the schema* to avoid a painful schema rewrite during migration. Update the `CREATE TABLE` to drop `NOT NULL` on the new columns.
- **`idx_match_canonical` UNIQUE conflict on existing data.** Some users may already have duplicate imports — a unique index creation will fail. Gate with phase 03's dedup step; do not add the unique index in `ensureAllTablesExist` unconditionally.
- **WAL on network filesystems.** Some users store DBs on Dropbox / NAS. WAL requires POSIX locking to work. Leave the PRAGMA as `WAL`; if user reports breakage, fall back to `journal_mode=DELETE` for that file only.
