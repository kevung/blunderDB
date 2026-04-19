# Follow-ups from the v2.0.0 optimization run

Items deferred from phases 00–06 that should be addressed in a future task sheet.

| # | Phase | Status | Item |
|---|---|---|---|
| 1 | 06 | open | `wails build` smoke test not run locally — GUI bindings and frontend build should be verified before tagging for release to users. |
| 2 | 06 | open | GUI manual smoke test on a real migrated production DB not done — exercise every filter in the Search modal to confirm no regressions. |
| 3 | 05 | done | Added `idx_position_pip_diff ON position(pip_diff)` and `idx_position_dice ON position(dice_1, dice_2)`. PipWindow benchmark unchanged (planner correctly scans when >50% rows match); dice index IS used for OR-dice queries. |
| 4 | 05 | open | `BenchmarkSearch_WinGammonCombo` (1221 ms) remains slow — bottleneck is the TEMP B-TREE sort after the analysis-driven join on the full tournois dataset. Added `idx_analysis_win_gammon (player1_win_rate, player1_gammon_rate)` but planner still uses single-column index. Root cause: sort cost, not lookup cost. |
| 5 | 02 | done | Added `idx_position_score_cube ON position(match_length, score_1, score_2, cube_value)` for score+cube combined filter. |
| 6 | 02 | done | Added `idx_game_match ON game(match_id)` — tournament subquery no longer SCANs game table. |
