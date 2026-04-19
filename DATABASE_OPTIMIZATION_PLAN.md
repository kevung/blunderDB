# blunderDB Database Optimization Plan

## Context

blunderDB is functionally complete but the SQLite schema was never tuned for the workloads it now sees: batch imports of tournament matches and filtered searches over thousands of positions. Two reported symptoms:

1. **Batch import is slow.** Every import preloads every existing position into RAM, deserializes each one, and inserts new rows without prepared statements. Time scales with the existing DB size, not just the file being imported.
2. **Filter searches are slow.** `LoadPositionsByFilters` issues `SELECT id, state FROM position` with **no WHERE clause**, decodes every JSON blob in Go, and then does an extra `LoadAnalysis(id)` per position for any analysis-related filter (win rate, gammon, equity error…). This is an O(N) JSON parse plus an O(N) N+1 query pattern.

Goal: make both common operations fast enough to handle a 10k+ position database (roughly the size of `testdata/tournois` once imported) without touching functionality. Decisions already made:

- **In-place auto-migration** 1.8.0 → 2.0.0 on `OpenDatabase`.
- **Add denormalized columns alongside JSON blobs** on `position` and `analysis`; keep the blobs intact.
- **Gate performance with Go benchmarks** (`benchmark_test.go`) run against a fixture built from `testdata/tournois/*.xg`.

All other docs I write (task sheets, follow-ups) must stay **≤500 lines** — add this rule to `CLAUDE.md` as part of implementation.

## Root causes

| # | Location | Problem |
|---|---|---|
| 1 | `db.go:1932` | `SELECT id, state FROM position` with no WHERE — every filter search fetches the whole table. |
| 2 | `db.go:1949` | `json.Unmarshal` on every row, even for cheap filters like `decision_type=cube`. |
| 3 | `db.go:1969–1991` | Analysis filters call `LoadAnalysis(p.ID)` **per position** → classic N+1. |
| 4 | `db.go:5805–5837` | Import preloads all existing positions into RAM map before inserting. |
| 5 | `db.go:savePositionInTx…` | No prepared statements inside import hot loop. |
| 6 | `db.go:36–65, 399–424` | No PRAGMA tuning (default `journal_mode=DELETE`, `synchronous=FULL`, no mmap). |
| 7 | `db.go` (no `CREATE INDEX`) | Missing indexes on `move(position_id)`, `move(game_id)`, `match(canonical_hash)` (unique), and every new filter column. |
| 8 | `db.go:importXXX` | `importCancelled` flag never checked inside the move loop. |

## Approach — schema (v2.0.0)

### New columns on `position`

All populated in Go at insert time; backfilled on migration. All `NOT NULL` except where noted.

```
zobrist_hash      INTEGER  -- 64-bit Zobrist XOR hash of the normalized position, UNIQUE
decision_type     INTEGER  -- 0 checker, 1 cube
player_on_roll    INTEGER
dice_1            INTEGER
dice_2            INTEGER
cube_value        INTEGER  -- nullable (money game / no cube)
cube_owner        INTEGER  -- nullable
score_1           INTEGER
score_2           INTEGER
match_length      INTEGER
has_jacoby        INTEGER
has_beaver        INTEGER
pip_1             INTEGER
pip_2             INTEGER
pip_diff          INTEGER  -- pip_1 - pip_2
off_1             INTEGER
off_2             INTEGER
back_checkers_1   INTEGER  -- checkers on points 18..24 for player 1
back_checkers_2   INTEGER  -- checkers on points 1..7 for player 2
no_contact        INTEGER  -- 0/1
occupancy_1       INTEGER  -- 26-bit bitboard: bit i set iff player 1 has ≥1 checker on point i
occupancy_2       INTEGER  -- 26-bit bitboard: bit i set iff player 2 has ≥1 checker on point i
point_mask_1      INTEGER  -- 26-bit bitboard: bit i set iff player 1 has ≥2 checkers (held point)
point_mask_2      INTEGER  -- 26-bit bitboard: bit i set iff player 2 has ≥2 checkers
```

Rationale: plain columns (not SQLite `GENERATED` columns) so INSERT paths stay explicit and the same logic backfills old rows.

### Chess-programming techniques applied

Two techniques from [chessprogramming.org](https://www.chessprogramming.org) translate cleanly to backgammon and are worth importing rather than inventing:

**Zobrist hashing** replaces what would otherwise be a SHA-256 of a canonical JSON string.

- Seed a fixed table of 64-bit random constants, checked into source, in a new `zobrist.go`: one `uint64` per `(point_index, checker_count, color)` tuple (26 points × ~16 checker counts × 2 colors ≈ 832 values), plus separate keys for every discrete state bit (player_on_roll, dice_1, dice_2, cube_value, cube_owner, score_1, score_2, match_length, has_jacoby, has_beaver, decision_type, bearoff counts).
- Hash = XOR of all contributing keys for a given position. **Computed directly from the `Board` / `Position` struct** — no JSON serialization in the hot path.
- Stored as a single SQLite `INTEGER` (8 B) with a `UNIQUE` index. Index nodes hold ~8× more keys per page than the 64-hex-char SHA-256 string would, so btree lookups during import dedup are faster and the file is smaller.
- Collision probability on 10⁶ positions is ≈2⁻⁴⁴ (birthday bound) — fine for dedup. The unique-index conflict surfaces any real collision as a normal SQL error, and `INSERT OR IGNORE` + `SELECT id WHERE zobrist_hash=?` handles it uniformly.
- The keys must be **deterministic across builds** (checked-in constants or `math/rand` with a fixed seed), otherwise previously-imported positions would rehash to new values and break dedup after an app upgrade. A unit test verifies hash stability for a handful of well-known positions.

**Bitboards** replace the per-point Go loop in `MatchesCheckerPosition` (model.go:230) and enable SQL-side pattern pre-filtering.

- `occupancy_1/2` and `point_mask_1/2` encode the board as 26-bit masks over SQLite `INTEGER`. Bitwise `&` and `|` are SQLite built-ins, so a filter like "player 1 holds a 5-prime across points 11–15" becomes `(point_mask_1 & 0x7C00) = 0x7C00` — one instruction per row, no JSON parse.
- The checker-structure filter in the Svelte UI compiles its template board into an `(occupancy_required, point_mask_required)` pair; SQL does the pre-filter, Go does the final exact-checker-count confirmation on the shrunk set. This is the bitboard equivalent of "cheap SQL pre-filter before expensive Go check" already mentioned in the search rewrite.
- Future pattern filters (blot on a specific point, anchor in home board, builder count in outer board) derive from these two masks — blots = `occupancy XOR point_mask`, etc. — without requiring further schema changes.

### New columns on `analysis`

```
best_cube_action         TEXT   -- "No double", "Double, take", etc.
cube_error               REAL   -- equity loss of played cube action
best_move_equity_error   REAL   -- equity loss of played checker move vs best
player1_win_rate         REAL
player1_gammon_rate      REAL
player1_backgammon_rate  REAL
player2_win_rate         REAL
player2_gammon_rate      REAL
player2_backgammon_rate  REAL
```

`analysis.data` JSON stays as the source of truth for detail views. All writes must go through a single `SaveAnalysis` helper that keeps columns and JSON in sync.

### New indexes

```
CREATE UNIQUE INDEX idx_position_zobrist ON position(zobrist_hash);
CREATE INDEX idx_position_decision_pip   ON position(decision_type, pip_diff);
CREATE INDEX idx_position_decision_dice  ON position(decision_type, dice_1, dice_2);
CREATE INDEX idx_position_off            ON position(off_1, off_2);
CREATE INDEX idx_position_score          ON position(match_length, score_1, score_2);
-- bitboards are not indexed: the selectivity of "& mask = mask" is too low for a
-- btree to help, and the filter runs after narrow SQL predicates have shrunk the set.
CREATE INDEX idx_analysis_position       ON analysis(position_id);
CREATE INDEX idx_analysis_win1           ON analysis(player1_win_rate);
CREATE INDEX idx_analysis_cube_error     ON analysis(cube_error);
CREATE INDEX idx_analysis_move_error     ON analysis(best_move_equity_error);
CREATE UNIQUE INDEX idx_match_canonical  ON match(canonical_hash);
CREATE INDEX idx_move_position           ON move(position_id);
CREATE INDEX idx_move_game               ON move(game_id);
```

Run `ANALYZE;` after backfill so the query planner gets stats.

### PRAGMAs (applied in `SetupDatabase` **and** `OpenDatabase`)

```
PRAGMA journal_mode = WAL;
PRAGMA synchronous  = NORMAL;
PRAGMA cache_size   = -65536;   -- 64 MB
PRAGMA temp_store   = MEMORY;
PRAGMA mmap_size    = 268435456; -- 256 MB
PRAGMA foreign_keys = ON;        -- already present
```

## Approach — import

Edit every `ImportXXXMatch` / `ImportXGPPosition` / `ImportBGFPosition` path in `db.go`:

1. **Delete the "load all positions into RAM" warmup** (`db.go:5805–5837` and analogues). The in-memory JSON-keyed map disappears entirely — SQLite's unique `zobrist_hash` index is the dedup oracle.
2. **Prepare statements once per transaction**, reuse in the move loop:
   - `INSERT OR IGNORE INTO position (zobrist_hash, state, decision_type, occupancy_1, …) VALUES (?, …)`
   - followed by `SELECT id FROM position WHERE zobrist_hash = ?` for the assigned id (handles dedup uniformly for new and existing rows). Both statements hit the unique index.
   - Prepared `INSERT INTO analysis (…)`, `INSERT INTO move (…)`, `INSERT INTO move_analysis (…)`.
   - Optional small per-transaction `map[uint64]int64` caches the lookup so duplicate positions within the same file skip even the index probe.
3. **Check `importCancelled`** with `atomic.LoadInt32` at the top of each game loop iteration so Ctrl+C / Cancel returns promptly.
4. **Checkpoint WAL after each file** in `cli.go:importBatch` (`PRAGMA wal_checkpoint(TRUNCATE)`) to keep the `-wal` file from ballooning during large batches.
5. Derived-column computation lives in one helper `populatePositionColumns(*Position)` callable from both normal insert and migration backfill. It calls into `zobrist.go` (hash), `epc.go` (pip counts), and a small `bitboards.go` (occupancy/point masks).

## Approach — search

Rewrite `LoadPositionsByFilters` (`db.go:1845`):

1. **Build SQL dynamically**. One query, `position p LEFT JOIN analysis a ON a.position_id = p.id`, WHERE clauses appended per active filter. Push these filters entirely to SQL:

   | Filter | SQL column |
   |---|---|
   | decision type | `p.decision_type` |
   | dice | `p.dice_1`, `p.dice_2` |
   | cube value | `p.cube_value` |
   | score 1/2 + match length | `p.score_1`, `p.score_2`, `p.match_length` |
   | has_jacoby / has_beaver | `p.has_jacoby`, `p.has_beaver` |
   | pip count (abs + diff) | `p.pip_1`, `p.pip_2`, `p.pip_diff` |
   | off 1/2 | `p.off_1`, `p.off_2` |
   | back checkers 1/2 | `p.back_checkers_1`, `p.back_checkers_2` |
   | no contact | `p.no_contact` |
   | win/gammon/backgammon (p1+p2) | `a.player1_win_rate`, … |
   | equity / cube / move error | `a.cube_error`, `a.best_move_equity_error` |
   | has-analysis | `a.id IS NOT NULL` |
   | match / tournament ids | `p.id IN (SELECT position_id FROM move WHERE game_id IN …)` (uses new indexes) |

2. **Bitboard pre-filter for checker structure.** When a `MatchesCheckerPosition` template is supplied, compile it once into `(occ_req_1, pt_req_1, occ_req_2, pt_req_2)` masks and append `(p.occupancy_1 & ?) = ? AND (p.point_mask_1 & ?) = ? AND …` to the WHERE clause. This shrinks the set to positions that could match before Go runs the exact per-point comparison.
3. **Keep in Go (after SQL shrinks the set)**: the exact per-point checker-count comparison for checker-structure filters (bitboards say "≥1 / ≥2", Go confirms the exact number), move-pattern regex, and mirror matching.
4. **Add `ORDER BY p.id` and pagination** (`LIMIT ?, OFFSET ?`) so the CLI `--limit` flag no longer needs to load-then-truncate.
5. **Return positions already joined with analysis** so callers don't do a second `LoadAnalysis(id)` per row.

## Migration (1.8.0 → 2.0.0)

In `db.go` (pattern at `db.go:430+`, extend the linear chain) and in `ensureAllTablesExist` (for idempotency on already-migrated DBs):

1. Bump `DatabaseVersion` → `"2.0.0"` in `model.go:36`.
2. Add a v2.0.0 migration step:
   - `ALTER TABLE position ADD COLUMN …` for each new column (nullable during migration).
   - `ALTER TABLE analysis ADD COLUMN …` for each new column.
   - Iterate every row: unmarshal JSON, call `populatePositionColumns` / `populateAnalysisColumns`, `UPDATE` in batches of ~1000 inside a single transaction.
   - Create indexes (after backfill to avoid slowing down UPDATEs).
   - `ANALYZE`.
   - Emit Wails progress events (`EventsEmit(a.ctx, "migration:progress", …)`) every N rows so the GUI can show a spinner/progress bar.
3. Skip everything if `position` table is empty (fresh DB).
4. Extend `migration_test.go` with a 1.8.0-snapshot fixture → verify every new column matches a recomputation, verify unique-hash constraint isn't violated, verify filter queries return the same positions as the old Go-side filter did.

## Files to modify

- `model.go` — bump `DatabaseVersion`; no struct changes (columns are internal to DB).
- `db.go` — schema (`SetupDatabase`, `ensureAllTablesExist`), PRAGMAs, migration block, `SavePosition`, `SaveAnalysis`, **full rewrite of `LoadPositionsByFilters`**, all `ImportXXX` hot paths, new helpers `populatePositionColumns`, `populateAnalysisColumns`, `buildSearchSQL`, `buildCheckerStructureMasks`.
- `cli.go` — no behavioural changes; let `runSearch` continue to call the new `LoadPositionsByFilters`. Add `PRAGMA wal_checkpoint(TRUNCATE)` at the end of `importBatch`.
- `epc.go` — expose a `PipCounts(board) (int, int)` helper if not already callable from `db.go`.
- **New**: `zobrist.go` — fixed 64-bit random key table (derived from a checked-in seed) + `ZobristHash(*Position) uint64` function + stability unit test.
- **New**: `bitboards.go` — `OccupancyMasks(*Board) (occ1, occ2, pt1, pt2 uint32)` and a `BuildCheckerStructureMasks(filter Position)` helper used by the search WHERE builder.
- `migration_test.go` — add 1.8.0 → 2.0.0 migration tests.
- **New**: `benchmark_test.go` at repo root.
- `CLAUDE.md` — add the 500-line doc-size rule and a short note on the v2.0.0 schema shape (Zobrist + bitboards).

## Benchmarks (`benchmark_test.go`)

Build the fixture once in a `TestMain` / `sync.Once` so each benchmark run reuses it:

```go
// setup: import every testdata/tournois/**/*.xg into an in-memory DB
// BenchmarkImport_TournoisCold       — fresh DB, import all files
// BenchmarkImport_TournoisIncremental — import one extra file into a full DB
// BenchmarkSearch_DecisionCube       — decision=cube
// BenchmarkSearch_ErrorAboveTenth    — cube_error>0.1 OR best_move_equity_error>0.1
// BenchmarkSearch_PipWindow          — pip_diff BETWEEN -2 AND 2
// BenchmarkSearch_WinGammonCombo     — player1_win_rate>0.55 AND player1_gammon_rate>0.2
// BenchmarkSearch_ScoreSpecific      — match_length=7, score_1=6, score_2=4
// BenchmarkSearch_DiceAndPlayer      — dice=(6,5) AND player_on_roll=0
// BenchmarkSearch_CheckerStructure   — bitboard pre-filter + exact Go compare
// BenchmarkSearch_PrimePattern       — bitboard-only: (point_mask_1 & prime) = prime
// BenchmarkZobristHash               — raw ZobristHash(*Position) throughput vs SHA-256 on JSON
```

Record baseline numbers **before** any schema change (by running against current code), then after. Targets (soft — document actuals in commit message):

- Cold import of `testdata/tournois/**/*.xg`: ≥3× faster.
- Each search benchmark: ≤100 ms on the full tournament fixture.

Run via `go test -bench=. -benchtime=5x ./...`.

## Verification

```bash
# 1. Baseline (current code, before any change)
go test -bench=. -benchtime=5x -run=^$ ./...    | tee bench-before.txt

# 2. Implement changes, then re-bench
go test -bench=. -benchtime=5x -run=^$ ./...    | tee bench-after.txt

# 3. Full test suite — no regressions
go test ./...
go test ./tests/...

# 4. Migration smoke test on a real 1.8.0 database
cp some-existing.db /tmp/pre-migration.db
./blunderdb list --db /tmp/pre-migration.db --type stats   # triggers migration
./blunderdb search --db /tmp/pre-migration.db --decision cube
./blunderdb search --db /tmp/pre-migration.db --error-min 0.1

# 5. GUI smoke test
wails dev
# open /tmp/pre-migration.db, exercise every filter in the Search modal
# confirm identical result counts vs pre-migration runs

# 6. CLI batch import benchmark (human-visible)
rm -f /tmp/tournois.db
time find testdata/tournois -name '*.xg' -print0 | \
  xargs -0 -n1 -I{} ./blunderdb import --db /tmp/tournois.db --type match --file {}
./blunderdb list --db /tmp/tournois.db --type stats
```

## Risks & mitigations

- **Long backfill on large DBs.** Emit Wails progress events; chunk UPDATEs (1000/row) inside one transaction; skip if position count is 0.
- **Drift between JSON blob and denormalized columns.** Mitigate by funneling every write through `SavePosition` / `SaveAnalysis`; add a debug assertion under `testing.Short()==false` that recomputes columns from JSON and compares.
- **Zobrist hash collisions.** 64-bit Zobrist gives ≈2⁻⁴⁴ collision probability at 10⁶ positions — acceptable. Unique index means any real collision surfaces as an `INSERT OR IGNORE` silently skipping a distinct position; guard with a follow-up `SELECT state WHERE zobrist_hash=?` equality check on the decoded struct before treating the found id as a dedup hit. Add a unit test with two logically-equivalent positions from different formats (XG vs SGF vs BGF) producing the same hash.
- **Zobrist key stability.** Keys must not drift between releases. Derive them from a fixed seed (`math/rand.New(rand.NewSource(0xB10DE4DB))`) generated at init, and freeze the 32 well-known positions' expected hashes in a table-driven test so any accidental constant change fails CI.
- **SQLite query planner surprises.** Run `ANALYZE` after backfill; spot-check a handful of search queries with `EXPLAIN QUERY PLAN` in a test.
- **Rollback path.** If 2.0.0 migration fails mid-way, the DB is left with partially-populated new columns. Wrap the whole v2.0.0 step in a transaction (SQLite supports transactional ALTER TABLE ADD COLUMN) — commit only if backfill + index creation both succeed.

## Out of scope

- Normalizing the per-candidate-move analysis (Option C). Revisit only if a future filter needs "Nth-best move equity loss".
- Rewriting imports for concurrency (parser-side).
- Any frontend changes — new SQL columns are internal; Svelte components keep receiving `Position` / `PositionAnalysis` structs unchanged.
- Incremental Zobrist updates after a move (chess engines XOR out/in per move for ~O(1) make/unmake). blunderDB doesn't search a game tree; each stored position is hashed once from scratch at insert, so the incremental-update machinery isn't worth building. Keep it in mind if a future feature needs live position-stream hashing.
