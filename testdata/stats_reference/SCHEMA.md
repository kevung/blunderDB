# Stats Reference JSON Schema

Each file in this directory captures the per-player statistics for one match as reported by eXtremeGammon (XG) and/or gnuBG. These files are loaded by `TestStatsParity` (fiche 01) as ground-truth data.

## Top-level fields

| Field | Type | Description |
|---|---|---|
| `match_file` | string | Relative path to the primary fixture file (XG or SGF) |
| `match_length` | int | Match length in points |
| `players` | [string, string] | Player names: [player1 (White/XG-P1), player2 (Black/XG-P2)] |
| `xg` | object\|null | Statistics as reported by eXtremeGammon. `null` when not available. |
| `gnubg` | object\|null | Statistics extracted from the gnuBG SGF GS tags. `null` when no SGF available. |
| `notes` | string | Free-text notes (encoding differences, sanity-check results, …) |

## Player stats object (inside `xg` or `gnubg`)

Both `xg` and `gnubg` have the same top-level shape: `{"player1": <PlayerStats>, "player2": <PlayerStats>}`.

### XG player stats

All equity errors are **positive magnitudes** (XG reports them as negative; we store magnitude for easier comparison).

| Field | Type | Description |
|---|---|---|
| `pr` | float\|null | Performance Rating (XG formula: 500 × \|equity_error\| / total_decisions) |
| `snowie_error_rate` | float\|null | Snowie Error Rate (500 × \|equity_error\| / total_moves_xg) |
| `total_decisions` | int\|null | XG's denominator: unforced checker + close cube decisions |
| `checker_unforced` | int\|null | Unforced checker move decisions (XG: excludes 1-legal-move positions) |
| `checker_forced` | int\|null | Forced checker moves (total - unforced); null when not reported by XG |
| `double_decisions` | int\|null | XG close doubling decisions |
| `take_decisions` | int\|null | Take decisions |
| `pass_decisions` | int\|null | Pass decisions |
| `total_errors` | int\|null | Positions classified as errors (any level) |
| `total_blunders` | int\|null | Positions classified as blunders |
| `total_equity_error_emg` | float\|null | Absolute total equity error (EMG, positive magnitude) |
| `checker_equity_error_emg` | float\|null | Absolute checker equity error |
| `double_equity_error_emg` | float\|null | Absolute doubling equity error |
| `take_equity_error_emg` | float\|null | Absolute take/pass equity error |
| `total_mwc_loss_pct` | float\|null | Total MWC loss in percentage points (positive magnitude) |
| `checker_mwc_loss_pct` | float\|null | Checker MWC loss |
| `double_mwc_loss_pct` | float\|null | Doubling MWC loss |
| `take_mwc_loss_pct` | float\|null | Take/pass MWC loss |

### gnuBG player stats

Derived from the `GS[M:…][C:…]` tags in the SGF using `gnubg/sgf.c:WriteStatContext` encoding. See `cmd/extract_gnubg_stats/main.go` for the extraction tool.

| Field | Type | Description |
|---|---|---|
| `checker_unforced` | int\|null | `anUnforcedMoves`: checker positions with >1 legal move |
| `checker_total` | int\|null | `anTotalMoves`: all checker positions |
| `checker_forced` | int\|null | `total - unforced`: positions with exactly 1 legal move |
| `checker_errors` | int\|null | VeryBad + Bad + Doubtful skill categories |
| `checker_blunders` | int\|null | VeryBad + Bad only |
| `checker_equity_error_emg` | float\|null | `arErrorCheckerplay[NORMALISED]`: total equity loss on checker plays |
| `checker_mwc_loss_pct` | float\|null | `arErrorCheckerplay[UNNORMALISED]` × 100: MWC loss on checker plays |
| `checker_pr_gnubg_1000` | float\|null | gnuBG-style checker PR (factor 1000): 1000 × equity / unforced |
| `checker_pr_xg_500` | float\|null | XG-style checker PR (factor 500): 500 × equity / unforced |
| `total_cube` | int\|null | `anTotalCube`: all cube positions |
| `double_decisions` | int\|null | `anDouble`: times the player doubled |
| `take_decisions` | int\|null | `anTake` |
| `pass_decisions` | int\|null | `anPass` |
| `cube_missed_double_dp` | int\|null | No-doubles below cash point that should have been doubled |
| `cube_missed_double_tg` | int\|null | No-doubles above too-good point |
| `cube_wrong_double_dp` | int\|null | Doubles below decision point |
| `cube_wrong_double_tg` | int\|null | Doubles above too-good point |
| `cube_wrong_take` | int\|null | Wrong takes |
| `cube_wrong_pass` | int\|null | Wrong passes |
| `cube_equity_error_emg` | float\|null | Sum of all cube equity errors (NORMALISED) |
| `cube_mwc_loss_pct` | float\|null | Sum of all cube MWC errors × 100 |
| `total_equity_error_emg` | float\|null | checker + cube equity error combined |
| `total_mwc_loss_pct` | float\|null | checker + cube MWC loss combined × 100 |
| `close_cube_decisions` | int\|null | `anCloseCube`: **NOT stored in SGF** — null unless obtained externally |
| `pr_gnubg` | float\|null | gnuBG combined PR — requires `close_cube_decisions`, null when unavailable |
| `snowie_er` | float\|null | Snowie Error Rate: `500 × total_equity_error_emg / (checker_total_P1 + checker_total_P2)`. Computed from extracted GS tag data; null when total_equity_error_emg is null (e.g. skill pass not run). Formula matches gnuBG formatgs.c:415-424 with factor 500 (XG-compatible). |

## Notes on sign convention

- All equity errors and MWC losses are stored as **positive magnitudes** in these files.
- XG reports them negative in its UI; this file normalises to positive for easier comparisons.
- The gnuBG `arErrorCheckerplay` values in the SGF are also stored as positive (gnuBG stores the absolute value of the error).

## Notes on gnuBG error factor

gnuBG's default `rErrorRateFactor` is **1000**; XG uses **500**. This means gnuBG PR ≈ 2 × XG PR for the same match. The field `checker_pr_xg_500` applies the factor 500 to the gnuBG data for apples-to-apples comparison with XG.

## Sanity check

For any match where both XG and gnuBG data are available:

```
xg.pr ≈ 500 × |xg.total_equity_error_emg| / xg.total_decisions  (±0.05)
gnubg.checker_pr_xg_500 ≈ 500 × gnubg.checker_equity_error_emg / gnubg.checker_unforced  (exact)
```
