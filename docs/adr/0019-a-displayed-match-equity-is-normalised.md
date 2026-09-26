# A displayed match equity is normalised — the MWC scale stays inside the engine

Status: accepted.
See also: ADR-0011, ADR-0016

## Context

gammonNet computes on two internal scales: `Decide` returns raw MWC, `Value`/`valueFromProbs`
return `2×MWC−1`. XG and gnubg display *normalised* equity (gnubg's `mwc2eq`), anchored on the
current cube: `eq = (2·mwc − cash − pass) / (cash − pass)`, ±1 = winning/losing the cube
outright. `2×MWC−1` coincides with it only at double match point; at 5-away/5-away it is 6.4×
too small. The column is shared with imported XG/GNUbg/BGBlitz analyses, which are normalised,
and the statistics read it as EMG — so an internal scale leaking out corrupts every PR, blunder
count and histogram, not only the display.

## Decision

1. **Every equity leaving `package gammonnet` is on one scale**: money points at money play,
   normalised equity at a match score.
2. **The conversion sits at the domain edge, not in the model.** `Decide` keeps returning MWC and
   `Value` keeps returning `2×MWC−1`: they are a faithful port judged against C gold, and a
   verdict or a ranking is invariant under any increasing affine map. `EvaluatePosition` and
   internal/gui's race path convert; nothing else does.
3. **One `EquityScale` per evaluated position** (`referential.go`), built from the position's own
   score and cube, with `FromDecision` (MWC) and `FromSearch` (`2×MWC−1`), because one panel
   row mixes both sources and only the caller knows which. Money play is the identity.
4. **Converting at the root is exact.** Within one search score and cube never move, the map is
   affine, and the opponent's anchors are (1−pass, 1−cash), so normalised equity negates ply by
   ply as MWC does. A position with no referential is **refused, never emitted unconverted** —
   an unscaled match equity is plausible-looking and wrong by a factor of six.
5. *(merged into rule 4.)*
6. **`TooGood` has its own label**: `"Too good to double, take"` / `"…, pass"`, the spelling
   `engine.BestCubeVerdict` decodes; `"No Double"` means only what it says.
7. Version label — see ADR-0011 rule 8.

## Consequences

- Money play is bit-identical; match-score numbers are on XG's scale.
- `BestCubeAction` has a fourth value, grouped verbatim by `storage/stats.go`, so "too good"
  is its own statistics row.
- Rejected: **converting inside `Decide`** (a presentation concern in the ported model, and the
  C gold would no longer hold); **keeping MWC with a relabelled header** (a header cannot make
  two units in one column comparable); **MWC percentages for moves** (blunder thresholds, PR and
  histograms are defined on an equity loss); **folding `TooGood` into "No Double"** (the two
  give opposite advice).

## Guard

`pkg/blunderdb/engine/gammonnet/referential_test.go` (identity at money, anchors at ±1 at every
score, coincidence with `2×MWC−1` at DMP, root conversion exact);
`internal/gui/gammonnet_eval_test.go` (refusal, not a fall to money).
