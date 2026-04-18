# Follow-ups from the v2.0.0 optimization run

Items deferred from phases 00–06 that should be addressed in a future task sheet.

| # | Phase | Item |
|---|---|---|
| 1 | 06 | `wails build` smoke test not run locally — GUI bindings and frontend build should be verified before tagging for release to users. |
| 2 | 06 | GUI manual smoke test on a real migrated production DB not done — exercise every filter in the Search modal to confirm no regressions. |
| 3 | 05 | `BenchmarkSearch_PipWindow` (228 ms) and `BenchmarkSearch_DiceAndPlayer` (122 ms) exceed the 100 ms target — consider adding a standalone `idx_position_pip_diff` index and a composite `idx_position_dice` index. |
| 4 | 05 | `BenchmarkSearch_WinGammonCombo` (1236 ms) is an outlier — the `ORDER BY p.id` sort after an analysis-driven join forces a full temp B-TREE on large result sets; investigate adding a covering index or pushing the ORDER BY into a subquery. |
| 5 | 02 | `idx_position_score` + `cube_value` query does a full SCAN (query plan #2) — add `idx_position_score_cube (score_1, score_2, cube_value)` if score+cube filtered searches are common in the GUI. |
| 6 | 02 | `idx_game_match (match_id)` missing on the `game` table — the match-IDs subquery SCANs game; low impact for small tournament filters but worth adding. |
