# 01 — Foundation utilities

**Goal:** Build the pure, DB-free helpers that imports, search, and migration will all call. Zero schema change in this phase.

**Depends on:** 00 (baseline captured).

## Files touched

- **New:** `zobrist.go`
- **New:** `zobrist_test.go`
- **New:** `bitboards.go`
- **New:** `bitboards_test.go`
- **Edit:** `epc.go` — export `PipCounts(board) (int, int)` if it isn't already callable from `db.go`.
- **Edit:** `db.go` — add (unused for now) `populatePositionColumns`, `populateAnalysisColumns` helpers near `SavePosition` / `SaveAnalysis`.

## 1. `zobrist.go` + test

### Key table

- [ ] Define one `uint64` per `(point_index ∈ 0..25, checker_count ∈ 0..15, color ∈ {Black,White})` → flat array `zobristPoint[26][16][2]`.
- [ ] Separate tables for discrete state:
  - [ ] `zobristPlayerOnRoll[2]`
  - [ ] `zobristDice[7][7]` — indices 1..6; `[0][0]` for "no dice".
  - [ ] `zobristCubeValue[11]` — values 1,2,4,…,1024 map to indices 0..10.
  - [ ] `zobristCubeOwner[3]` — Black, White, None.
  - [ ] `zobristScore1[64]`, `zobristScore2[64]`, `zobristMatchLength[64]` — 0..63 covers every realistic match.
  - [ ] `zobristFlags` — single `uint64` with helpers for `has_jacoby`, `has_beaver`, `decision_type`.
  - [ ] `zobristBearoff[2][16]` — `(color, count)`.
- [ ] Populate from a **fixed seed** with `math/rand.New(rand.NewSource(0xB10DE4DB))` in `init()`. Never change the seed or the population order.

### `ZobristHash(*Position) uint64`

- [ ] XOR together every contributing key based on the `Position` struct.
- [ ] **Normalize `player_on_roll` to 0** before hashing: if `p.PlayerOnRoll == 1`, mirror the board (swap colors, swap bearoff slots) into a scratch copy, then hash. Matches the existing `savePositionInTxWithCache` normalization (db.go:6688–6729).
- [ ] Ignore fields that don't belong in the identity (e.g. `Position.ID`).

### Stability test (`zobrist_test.go`)

- [ ] Table-driven test with **32 well-known positions** (initial position, bar checker, bearoff endgame, a double-hit, money game, various cube/score/match-length combos, race positions, blitz, prime vs prime, back game, etc.). Freeze their expected `uint64` hashes in the test — any accidental constant change in the key table fails CI.
- [ ] Equivalence test: a position with `PlayerOnRoll=0` and its mirror with `PlayerOnRoll=1` hash to the **same** value.
- [ ] Cross-format test: import the same match twice (once from `testdata/test.xg`, once from `testdata/test.sgf` if the same game exists in both), pull two equivalent positions, assert equal hashes.

## 2. `bitboards.go` + test

### `OccupancyMasks(*Board) (occ1, occ2, pt1, pt2 uint32)`

- [ ] Walk all 26 `Board.Points`; set bit `i` in the right mask based on `(color, checkers)`:
  - `checkers ≥ 1` → `occupancy_{color}`
  - `checkers ≥ 2` → `point_mask_{color}`
- [ ] Return as `uint32` (26 bits fit easily). Store into SQLite via `int64` later.

### `BuildCheckerStructureMasks(filter Position) (...)`

- [ ] Compile a template `Position` (the UI's "search by board pattern" filter) into `(occ_req_1, pt_req_1, occ_req_2, pt_req_2)`:
  - For each point where the template says "≥N checkers of color C", set the bit in the corresponding `occ` mask; if N≥2, also set it in `pt`.
- [ ] Return the four masks plus a boolean `tight` saying whether any further Go-side check is needed (e.g. exact counts above 2).

### Tests (`bitboards_test.go`)

- [ ] Mask of the initial position matches a hand-computed expected value.
- [ ] A 5-prime on points 11–15 is detected by `(point_mask_1 & 0x7C00) == 0x7C00`.
- [ ] `BuildCheckerStructureMasks` round-trips a template: `OccupancyMasks(template) & mask_req == mask_req`.

## 3. `epc.go` surface

- [ ] Grep for `PipCount` / `PipCounts` in `epc.go`. If it's unexported or only callable as a method on a `BearoffDatabase`, add a top-level helper `PipCounts(board Board) (pip1, pip2 int)` that doesn't need the bearoff DB (the pip count is purely positional).

## 4. Column populators in `db.go`

Add (but do not yet call from any INSERT):

- [ ] `populatePositionColumns(p *Position) positionColumns` returning a struct of every new column's value (`zobrist_hash`, `decision_type`, `dice_1`, `dice_2`, `pip_1`, `pip_2`, `pip_diff`, `off_1`, `off_2`, `back_checkers_1`, `back_checkers_2`, `no_contact`, `occupancy_1`, `occupancy_2`, `point_mask_1`, `point_mask_2`, plus cube/score/jacoby/beaver mirrors).
- [ ] `populateAnalysisColumns(a *PositionAnalysis, playedMove, playedCubeAction string) analysisColumns` returning `best_cube_action`, `cube_error`, `best_move_equity_error`, `player1_win_rate`, …, `player2_backgammon_rate`.
  - Win/gammon/backgammon come from `a.DoublingCubeAnalysis.PlayerWinChances` etc. (see `model.go:143` — note semantics: "player" = player on roll = player 1 for player-1 columns).
  - `cube_error` = equity loss of the **played** cube action vs `BestCubeAction` (both present in `DoublingCubeAnalysis`).
  - `best_move_equity_error` = equity loss of the **played** checker move vs `CheckerAnalysis.Moves[0]` (ranked best first in most sources). If no played move is recorded, leave `NULL`.
- [ ] Unit tests in an existing or new `_test.go` file: feed a known `Position` + analysis, assert every returned field.

## Acceptance criteria

- [ ] `go test ./... -run 'Zobrist|Bitboard|Populate'` is green.
- [ ] `go test ./...` still green (no regressions in existing tests).
- [ ] `go build ./...` compiles; no unused-variable warnings from the populators (they're allowed to be dead code until phase 02).
- [ ] `wc -l zobrist.go bitboards.go` each well under 500 lines.

## Risks

- **Key-table size.** The `[26][16][2]` `uint64` table is ~6.5 KB — trivial, but declare it with a fixed-size array, not a slice, so the linker keeps it in `.rodata`.
- **Mirror-for-hash bugs.** Mirroring 25→0 bar and 0→25 bar is the classic off-by-one in backgammon code. Add a targeted mirror unit test.
- **`player_1` vs "player on roll" confusion.** `Position.PlayerOnRoll` flips each turn; `PositionAnalysis.DoublingCubeAnalysis.PlayerWinChances` is always from the on-roll POV. The analysis columns should follow the on-roll convention (document this in a one-line comment in `populateAnalysisColumns`).
