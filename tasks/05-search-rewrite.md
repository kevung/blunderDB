# 05 — Search rewrite

**Goal:** `LoadPositionsByFilters` pushes every cheap filter to SQL in one JOINed query; checker-structure filters use a bitboard pre-filter; analysis data arrives already joined; pagination is first-class.

**Depends on:** 03 (schema + migration populated all denormalized columns). Can run in parallel with 04 once 03 lands.

## Files touched

- `db.go` — rewrite `LoadPositionsByFilters` (entry at db.go:1845), add `buildSearchSQL` helper, drop N+1 analysis fetch.
- `cli.go` — minor: remove post-fetch filtering for `errorMin` / `hasAnalysis` (now done in SQL). Keep public `runSearch` signature stable.

## 1. Split the current function

The current `LoadPositionsByFilters` (db.go:1845) does three things: parse filter strings, fetch-and-filter positions, return `[]Position`. Split into:

- [x] `parseFilters(args) (sqlFilters, goFilters, error)` — implemented as `parseIntFilterExpr` / `parseFloatFilterExpr` + `appendIntRangeSQL` / `appendFloatRangeSQL` helpers.
- [x] `buildSearchSQL(f sqlFilters) (query string, args []any)` — inlined in `loadPositionsByFiltersCore`.
- [x] `runSearchQuery(query, args) ([]Position, []*PositionAnalysis, error)` — the scan loop in `loadPositionsByFiltersCore`.
- [x] `applyGoFilters(positions, analyses, f goFilters) ([]Position, []*PositionAnalysis)` — `matchesGoFilters` closure inside the row loop.

## 2. SQL query shape

One query, LEFT JOIN so `hasAnalysis = false` filters still return positions without an analysis row:

```sql
SELECT p.id, p.state,
       a.id, a.data,
       a.cube_error, a.best_move_equity_error,
       a.player1_win_rate, a.player1_gammon_rate, a.player1_backgammon_rate,
       a.player2_win_rate, a.player2_gammon_rate, a.player2_backgammon_rate,
       a.best_cube_action
FROM position p
LEFT JOIN analysis a ON a.position_id = p.id
WHERE <dynamic>
ORDER BY p.id
LIMIT ? OFFSET ?;
```

### Filter → SQL mapping

| Filter (current Go code) | SQL |
|---|---|
| `MatchesDecisionType` | `p.decision_type = ?` |
| `MatchesDiceRoll` | `p.dice_1 = ? AND p.dice_2 = ?` (order-insensitive: add `OR swapped`) |
| cube value | `p.cube_value = ?` or `IS NULL` for money |
| score + match length | `p.match_length = ? AND p.score_1 = ? AND p.score_2 = ?` |
| has_jacoby / has_beaver | `p.has_jacoby = ?` / `p.has_beaver = ?` |
| pip diff (`-2..2`) | `p.pip_diff BETWEEN ? AND ?` |
| absolute pip (p1) | `p.pip_1 BETWEEN ? AND ?` |
| off_1 / off_2 | `p.off_{1,2} >= ?` |
| back_checkers_{1,2} | `p.back_checkers_{1,2} >= ?` |
| no_contact | `p.no_contact = 1` |
| player-on-roll | `p.player_on_roll = ?` |
| win/gammon/backgammon rate (p1) | `a.player1_win_rate BETWEEN ? AND ?` etc. |
| same for p2 | `a.player2_*` |
| equity / cube error | `a.cube_error >= ?` |
| move error | `a.best_move_equity_error >= ?` |
| has_analysis | `a.id IS NOT NULL` |
| match IDs | `p.id IN (SELECT m.position_id FROM move m WHERE m.game_id IN (SELECT id FROM game WHERE match_id IN (?, ?, …)))` |
| tournament IDs | outer IN expands via the match table |

- [x] Parameter binding only — never string-concat filter values into SQL.
- [x] When a filter is absent, its clause is simply not appended; avoid `AND ? IS NULL` tricks.

## 3. Checker-structure pre-filter

When `filter.Board` has any non-zero points:

- [x] Call `bitboards.BuildCheckerStructureMasks(filter)` → `(occ1Req, pt1Req, occ2Req, pt2Req, tight)`.
- [x] Append: `AND (p.occupancy_1 & ?) = ? AND (p.point_mask_1 & ?) = ? AND (p.occupancy_2 & ?) = ? AND (p.point_mask_2 & ?) = ?`.
- [x] If `tight == false` (i.e. the template only required "≥1" or "≥2"), skip Go-side verification — SQL is sufficient.
- [x] If `tight == true` (template required an exact count ≥3 somewhere), keep the existing Go-side `MatchesCheckerPosition` loop (model.go:230) — but now it runs on a tiny set.

## 4. Remaining Go-side filters

- [x] Move-pattern regex (`MatchesMovePattern`, db.go:3161) — requires `LoadAnalysis` JSON content.  The analysis is already in hand from the JOIN → parse the `data` JSON once, apply regex, keep or drop.
- [x] Mirror matching — unchanged, works on decoded `Position`.
- [x] The `pos.MatchesMoveErrorFilter` path previously called `getPlayer1MovesForPosition` per row. Move-error is already covered by `a.best_move_equity_error`. Delete the per-row `move` query.

## 5. Ordering & pagination

- [x] `ORDER BY p.id` (deterministic, matches existing behavior).
- [x] `LIMIT ? OFFSET ?` — accept zero `limit` to mean "all" (don't emit `LIMIT` in that case).
- [x] Update CLI `--limit` plumbing (`cli.go:1867`) to pass the limit into the SQL, dropping the post-fetch `filteredPositions[:limit]` slice (cli.go:1952).

## 6. Backwards-compatible signature

- [x] Keep the 33-parameter `LoadPositionsByFilters` signature — it's bound to Wails and the CLI. Internally it immediately calls `loadPositionsByFiltersCore`.
- [ ] Tomorrow's refactor: introduce a `type SearchRequest struct` and deprecate the long arg list. Out of scope here.

## 7. Tests

- [x] **Equivalence tests.** For each filter category, write a test that:
  - Seeds an in-memory DB from `testdata/test.xg` + an XGP.
  - Runs the filter through both the old implementation (tag-build: checkout the pre-rewrite function as `legacyLoadPositionsByFilters`) and the new one.
  - Asserts identical sorted `[]Position.ID`.
  - Keep the `legacy*` copy in a `_test.go` file so it's not shipped.
- [x] `TestSearch_PaginationStable` — the first 10 of a 100-row result equals (page1 + page2) from `LIMIT 5 OFFSET 0` / `LIMIT 5 OFFSET 5`.
- [x] `TestSearch_PrimePattern_BitboardOnly` — a clean 5-prime template returns the same rows whether the Go-side verification is on or off.

## Acceptance criteria

- [x] `go test ./...` and `go test ./tests/...` green.
- [ ] Every search benchmark from sheet 00 meets the ≤100 ms target on the `testdata/tournois` fixture.
- [ ] `BenchmarkSearch_PrimePattern` (bitboard-only) is the fastest — reads only the integer mask columns.
- [ ] `BenchmarkSearch_CheckerStructure` (bitboard pre-filter + exact Go compare) is ≥5× faster than baseline.
- [ ] The Svelte Search modal, exercised manually on a migrated real DB, returns identical result counts to pre-v2.0.0 for every filter.

## Risks

- **Filter-string parsing drift.** The existing string filters (`"0.1..0.3"`, `">5"`) are parsed inside the filter methods on `Position`. Move the parsing into `parseFilters` and unit-test each operator. One test per operator.
- **Dice order.** The UI sometimes stores dice as `(1,3)` and sometimes `(3,1)`. Normalize on insert (phase 02 already does via `populatePositionColumns` — ensure `dice_1 <= dice_2`). Then SQL becomes a simple `=` comparison. Update the populator if it doesn't already sort.
- **NULL-safe equality.** `cube_value = NULL` is always false; use `cube_value IS NULL` when filtering money-game positions.
- **LEFT JOIN cardinality.** If `analysis` grew to multiple rows per position in any legacy DB, the LEFT JOIN multiplies result rows. `SELECT DISTINCT p.id` in an outer query, or add a `LIMIT 1` subselect, if sheet 03's dedup doesn't catch it. Spot-check on a migrated real DB.
- **`EXPLAIN QUERY PLAN`.** Before merging, run `EXPLAIN QUERY PLAN` against 4–5 representative queries and confirm the planner uses the new indexes. Commit the output to `tasks/search-query-plans.txt` for reference.
