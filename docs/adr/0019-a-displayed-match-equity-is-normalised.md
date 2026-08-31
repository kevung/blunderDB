# A displayed match equity is normalised — the MWC scale stays inside the engine

## Status

accepted — decided 2026-08-31. **Amends ADR-0016**: its "Consequences" already stated the goal
("normalised equity spans [−1, +1] … this matches XG and gnubg"), and the implementation did not
meet it. Amends the referential sentence ADR-0017 states twice ("2×MWC−1 at a match score");
ADR-0018's single reference to the referential stays true as written. Supersedes nothing; ADR-0011's boundary rule and ADR-0013's gap rule are untouched.

Also corrects an upstream comment: `gammonNet/src/gn_met.h` presents `2 * mwc - 1` as "the
equivalent-to-money scale that engines print", which is where the mistake entered blunderDB.

## Context

The Eval panel was reported as giving wrong cube equities and never finding a "too good". Both
were real, and the first is the larger of the two.

**Three scales in one panel.** At a match score gammonNet's numbers reached the user on whichever
internal scale produced them:

| what | function | scale it was printed in |
| --- | --- | --- |
| candidate moves, cubeless fact | `valueFromProbs` / `CubelessValue` | 2×MWC−1 |
| no-double / double-take / double-pass | `Decide` (`cube.go`) | raw MWC, in [0, 1] |
| money, everything | — | points (correct) |

`Value` — the search's leaf — does apply `2*x−1` to `levelBlend`; `Decide` never did, and
`evaluateCube` wrote its output straight into `CubefulNoDoubleEquity` while its own comment
claimed the result was "2×MWC−1 at a match score". The opening position, 0-ply:

| | cubeless | no double | double/take | double/pass |
| --- | --- | --- | --- | --- |
| money | +0.040 | +0.056 | −0.255 | +1.000 |
| 5-away/5-away, as displayed | +0.006 | **+0.504** | +0.480 | **+0.577** |
| 3-away/7-away, as displayed | **+0.559** | **+0.767** | +0.711 | +0.842 |
| 5-away/5-away, normalised | +0.039 | +0.055 | −0.256 | +1.000 |

**Neither internal scale is what engines print.** XG and gnubg display *normalised* equity —
gnubg's `mwc2eq` — anchored on the current cube: winning its value outright is +1, losing it
outright is −1,

    eq = (2·mwc − cash − pass) / (cash − pass)

`2×MWC−1` is that formula with cash = 1 and pass = 0, i.e. anchored on winning or losing the whole
*match*. The two coincide at double match point and nowhere else: at 5-away/5-away the search
scale is 6.4× too small, and at a lopsided score it is shifted as well, since cash + pass is no
longer 1.

**This is not only a display defect.** `CubefulNoDoubleEquity`, `CheckerMove.Equity` and
`EquityError` are the same columns an imported XG, GNUbg or BGBlitz analysis is stored in, and
those are normalised. blunderDB's own statistics read that column as EMG
(`engine.ConvertEMGLossToMWCLoss`, which converts an EMG loss *to* a MWC loss — it presupposes the
column is not already one). So a batch-analysed match position contributed errors 6× too small to
every PR, blunder count and error histogram, next to XG rows in the correct unit.

**And the "too good" verdict was computed and thrown away.** `Verdict` returns all four actions,
`Decide` propagates `TooGood`, `race.VerdictTooGood` renders it on the race panel — but
`cubeActionLabel` mapped `TooGood` onto `"No Double"`, deliberately, for want of a fourth label.
A position that is too good to double and one that is not good enough to double therefore read
identically, which is the opposite of what the two verdicts say. The vocabulary was already in
the codebase on both sides: `engine.BestCubeVerdict` decodes `"Too good to double, take"` (and
warns that "too good" is the trap that contains the word "double"), and the frontend already
translates `epc.race.verdicts.too_good` into all nine languages.

## Decision

1. **Every equity leaving `package gammonnet` is on one scale**: money points at money play,
   normalised equity at a match score. That is the scale the rest of the application already
   speaks — the imported analyses in the same column, the statistics reading it, the panel
   showing it beside them.

2. **The conversion sits at the domain edge, not in the model.** `Decide` keeps returning MWC and
   `Value`/`valueFromProbs` keep returning 2×MWC−1: they are a faithful port of `gn_cube.c`, they
   are judged against C gold files (`cube_gold_test.go`), and a verdict or a move ranking is
   invariant under any increasing affine map of the MWC anyway. `EvaluatePosition` and
   internal/gui's race bonus convert; nothing else does.

3. **One `EquityScale` per evaluated position** (`referential.go`), built from the position's own
   score and cube, with two methods that name where the number came from — `FromDecision` (MWC)
   and `FromSearch` (2×MWC−1) — because the panel mixes both sources in one row and only the
   caller knows which is which. Money play is the identity.

4. **Converting at the root is exact, not an approximation.** Within one search the score and cube
   never move, the map is affine, and the opponent's anchors are (1−pass, 1−cash) — so normalised
   equity negates ply by ply exactly as the MWC scale does. Converting each leaf and converting
   the root agree. `referential_test.go` holds all four properties, including the anchors landing
   on ±1 at every score and the coincidence with 2×MWC−1 at DMP.

5. **A position with no referential is refused, never emitted unconverted.** An unscaled match
   equity is plausible-looking and wrong by a factor of six — the failure mode this whole ADR is
   about.

6. **`TooGood` gets its label back**: `"Too good to double, take"` / `"…, pass"`, the spelling
   `engine.BestCubeVerdict` already decodes, with the suffix naming what the opponent would do if
   the cube came anyway. `"No Double"` now means only what it says.

7. **`EngineVersion` becomes `gammonNet v1.2.0`**, so `AnalyzeStaleGammonNet` re-runs the
   analyses this engine wrote itself. Imported analyses are untouched (ADR-0013).

## Considered options

**Convert inside `Decide`.** Rejected: it puts a presentation concern inside the ported model,
and invalidates `cube_gold_test.go` against the C reference — the port would no longer be a port.
The same argument, one level up, is why `Value` keeps its own scale too.

**Keep the MWC and change the column headers to say so.** Rejected: the column is shared with
imported XG/GNUbg analyses that are normalised. A header cannot make two units in one column
comparable, and every statistic reading that column would still be wrong.

**Print match winning chance as a percentage everywhere, including for moves.** Rejected for the
same reason plus a second one: an equity *loss* is what a blunder threshold, a PR and an error
histogram are defined on, and a MWC difference is not that quantity.

**Do nothing about `TooGood`, keep folding it into "No Double".** Rejected: the two verdicts give
opposite advice about playing on, the distinction already exists everywhere else in the codebase
(`domain.TooGood`, `race.VerdictTooGood`, `BestCubeVerdict`, nine translations), and only this
one string was discarding it.

## Consequences

- **Displayed numbers change at every match score, again** — this time towards XG's, which is
  what ADR-0016 said would happen the first time. Money play is bit-identical.
- **Stored gammonNet analyses at a match score were wrong and are re-run**, by the existing
  staleness path, on the version bump. Nothing else in the database moves.
- **The statistics computed over gammonNet-analysed match positions change**: errors that were
  6× too small become correct, so PR figures over such positions rise.
- **`BestCubeAction` gains a fourth value.** It is grouped verbatim by the statistics
  (`storage/stats.go`), so "too good" becomes its own row there — correct, and the reason the
  spelling matches the one XG imports already write rather than inventing a shorter label.
- **The verdict cell can now hold a sentence**, so it wraps instead of widening the table
  (ADR-0018: the Eval panel has no room to scroll).
- **A C-side comment is wrong upstream** and is corrected in the gammonNet repository. The C code
  itself is right: its spec (§5) works on the MWC scale deliberately, and it never claims to
  print an equity.
