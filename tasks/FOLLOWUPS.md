# Follow-ups from the v2.0.0 optimization run

Items deferred from phases 00–06 that should be addressed in a future task sheet.

| # | Phase | Status | Item |
|---|---|---|---|
| 1 | 06 | done | `wails build` is now exercised on every tagged release: the CI matrix builds all 4 platforms (ubuntu-latest webkit2gtk-4.1, ubuntu-22.04 webkit2gtk-4.0, windows, macos) and `wails dev` compiles the same Go+frontend path locally. Verified green on the 0.26.1 tag build (run 27267855792). |
| 2 | 06 | open | GUI manual smoke test on a real migrated production DB not done — exercise every filter in the Search modal to confirm no regressions. Low priority; covered indirectly by the searchFilterService unit + round-trip tests (see `frontend/src/__tests__/searchFilterService.test.js`). |
| 3 | 05 | done | Added `idx_position_pip_diff ON position(pip_diff)` and `idx_position_dice ON position(dice_1, dice_2)`. PipWindow benchmark unchanged (planner correctly scans when >50% rows match); dice index IS used for OR-dice queries. |
| 4 | 05 | known | `BenchmarkSearch_WinGammonCombo` (1221 ms) — known, low impact. Bottleneck is the TEMP B-TREE sort after the analysis-driven join on the full tournois dataset; `idx_analysis_win_gammon (player1_win_rate, player1_gammon_rate)` exists but the planner still uses a single-column index. Root cause is sort cost, not lookup cost; only this one combined win/gammon query is affected. Revisit only if it becomes user-visible (fix = query restructuring). |
| 5 | 02 | done | Added `idx_position_score_cube ON position(match_length, score_1, score_2, cube_value)` for score+cube combined filter. |
| 6 | 02 | done | Added `idx_game_match ON game(match_id)` — tournament subquery no longer SCANs game table. |
