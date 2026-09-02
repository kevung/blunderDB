# Architecture decision records

One file per decision, numbered in the order they were taken. A record is never
rewritten once accepted: a later decision that changes it says so in its own
*Status* section ("amends", "extends", "completes", "supersedes"), and this index
mirrors those statements. Read the relevant record before touching the subsystem
it governs — `CLAUDE.md` lists the invariants that come out of them.

*Date* is the decision date stated in the record's *Status* section; records
0001–0006 carry none, so the date of the commit that introduced them is used.

| # | Title | Status | Date | Relations | Context |
|---|---|---|---|---|---|
| [0001](0001-individually-imported-is-a-sticky-flag.md) | Individually-imported provenance is a sticky flag | accepted | 2026-07-14 | — | A position brought in on its own keeps that flag even when a later match import contains it; the flag is not part of the Zobrist identity. |
| [0002](0002-gui-position-write-goes-through-one-backend-call.md) | Saving the board is one backend call | accepted | 2026-07-14 | — | The GUI stops checking whether a position exists before writing; the backend decides insert vs. update in one call. |
| [0003](0003-search-picks-matches-tournaments-gui-keeps-ids-elsewhere.md) | GUI search picks matches and tournaments from lists | accepted | 2026-07-15 | — | Free-text IDs in the search panel became a picker modal; the CLI and the command line keep raw IDs. |
| [0004](0004-host-capabilities-are-detected-and-fall-back-never-assumed.md) | Host capabilities are detected and fall back | accepted | 2026-07-16 | — | Fonts, clipboard and the last-opened database are probed at runtime and degrade gracefully instead of being assumed present. |
| [0005](0005-serve-daemon-delegates-authentication.md) | The serve daemon delegates authentication | accepted | 2026-07-17 | — | `blunderdb serve` trusts `X-Tenant-ID` and must sit behind an authenticating reverse proxy; auth is never added to the engine. |
| [0006](0006-source-tool-study-marks-are-position-properties.md) | Source-tool study marks are position properties | accepted | 2026-07-26 | — | XG flags are a sticky property of the position, ORed on import, not an auto-filled collection. |
| [0007](0007-watermarks-mark-origin-and-nothing-else.md) | Watermarks mark origin and nothing else | accepted | 2026-08-03 | supersedes its own unreleased first version | A signed origin mark is written by the producer at export; nothing is ever recorded on the recipient's side (no registry, log or lineage). |
| [0008](0008-one-type-scale-and-controls-inherit-it.md) | One type scale, and form controls inherit it | accepted, applied | 2026-08-04 | — | Components use the `--font-size-*` tokens only; `font: inherit` on controls is global since 2026-08-11; named exceptions carry their own token. |
| [0009](0009-race-win-chances-are-read-or-convolved-cube-verdicts-are-never-estimated.md) | Race win chances read or convolved; verdicts never estimated | accepted | 2026-08-09 | extended by 0012; display refined by 0017 | Two-sided bearoff tables give exact win chances and money cube verdicts; outside them a calibrated convolution estimates p with a bound, and no verdict is estimated. |
| [0010](0010-luck-is-a-fact-of-the-move-never-recomputed.md) | Luck is a fact of the move, never recomputed | accepted | 2026-08-26 | — | Luck is read from the source file at import and stored; blunderDB does not recompute it. |
| [0011](0011-gammonnet-is-ported-to-go-and-the-representation-boundary-sits-at-the-evaluator-s-edge.md) | gammonNet is ported to Go; representation boundary at the evaluator | accepted | 2026-08-28 | — | The evaluator is a Go port of gammonNet compiled into the binary; conversion to its board representation happens once, at its edge. |
| [0012](0012-a-race-has-three-regimes-exact-evaluated-and-never-estimated.md) | A race has three regimes: exact, evaluated, never estimated | accepted | 2026-08-28 | extends 0009 | Beyond the two-sided domain the neural evaluator supplies the cube verdict (including at a match score); the static-model estimate still never does. |
| [0013](0013-evaluations-fill-gaps-an-imported-analysis-is-never-overwritten.md) | Evaluations fill gaps; imports are never overwritten | accepted | 2026-08-28 | narrow exception granted by 0016 | The embedded evaluator analyses positions that have no analysis; an XG/GNUbg/BGBlitz analysis is never replaced. |
| [0014](0014-the-integration-gate-is-judged-by-xg-not-by-the-gnubg-fixtures.md) | The integration gate is judged by XG | accepted | 2026-08-28 | — | The port's acceptance test compares against XG analyses on real positions, not against the gnubg fixtures used for parity. |
| [0015](0015-blunderdb-serve-operates-on-a-library-it-does-not-expose-an-evaluator.md) | `serve` operates on a library, not an evaluator | accepted | 2026-08-28 | — | The daemon may evaluate to fill its own library but exposes no generic evaluation endpoint; gammonGo's API contract is unchanged. |
| [0016](0016-the-referential-is-a-property-of-the-position-and-the-search-honours-it.md) | The referential is a position property the search honours | accepted | 2026-08-31 | amended by 0019; depended on by 0017 | Money play vs. match score is decided by the position; the search runs `use_match`, `EngineVersion` names a real gammonNet tag, `use_cube` deferred. |
| [0017](0017-the-panel-shows-position-facts-plus-the-one-decision-the-board-asks.md) | The panel shows position facts plus the one decision | accepted, applied | 2026-08-31 | refines display of 0009 and 0012; amended by 0018 and 0019; extended by 0021 | The Eval panel separates facts (win/gammon/backgammon chances, equity) from the single decision the board asks — checker play or cube — never both. |
| [0018](0018-the-axis-of-a-fact-follows-the-list-it-is-read-against.md) | The axis of a fact follows the list it is read against | accepted | 2026-08-31 | amends 0017 (rules 1, 2, 5); completed by 0020 | The pre-roll vector takes the axis of the candidate list it heads; per-side facts stay per-side; one table idiom across the panel. |
| [0019](0019-a-displayed-match-equity-is-normalised.md) | A displayed match equity is normalised | accepted | 2026-08-31 | amends 0016 and 0017; relied on by 0020 | One equity scale leaves the engine — money points or normalised equity (±1 = the current cube); MWC and 2×MWC−1 stay internal; "too good" gets its label back. |
| [0020](0020-a-cube-decision-has-one-shape-whatever-regime-produced-it.md) | A cube decision has one shape, whatever regime produced it | accepted | 2026-08-31 | completes 0018 rule 5; lands on 0019; extended by 0021 | Exact, evaluated and imported cube verdicts are rendered by the same component with the same four-way verdict and a single provenance strip. |
| [0021](0021-position-facts-are-two-stacked-blocks-and-slack-is-never-spread.md) | Position facts are two stacked blocks on one grid | accepted | 2026-09-01 | extends 0017 rule 1 and 0020 rule 8 | Layout only: facts render as two blocks sharing one column grid; spare width accumulates at the edge, never between blocks. |
| [0022](0022-the-live-cube-curve-runs-to-the-average-win-not-to-the-cash-equivalent.md) | The live cube curve runs to the average win | accepted | 2026-09-01 | fixes the implementation 0019 and 0020 relied on; ported upstream (gammonNet v1.2.0) | The Janowski curve had two segments clamped to the cash equivalent, which made "too good" unreachable; the tails now run to (0, −L) and (1, +W). |
| [0023](0023-the-search-values-its-leaves-with-the-cube.md) | The search values its leaves with the cube | accepted | 2026-09-02 | closes the follow-up 0016 point 7 left open; lands on 0019 and 0022; pinned to gammonNet v1.2.1 | Every leaf of the checker-move search is priced through the cube model at the position's own cube state (upstream `use_cube`), which is where gammon-go and gammon-save at a score actually come from; the probability walk swaps the score at every ply, as the scalar one always did. |

## Conventions

- Filenames are `NNNN-<slug>.md`; the slug is the decision in one sentence.
- Sections: *Status*, *Context*, *Decision*, *Considered options*, *Consequences*.
  *Status* carries the date and every relation to other records.
- Numbers are allocated when the record lands on `main`, not when it is drafted
  (0022 was drafted as 0021 and renumbered; nothing in a record depends on its number).
- Adding a record: append a row here in the same commit.
