# The panel shows position facts, plus the one decision the board asks

Status: accepted.
See also: ADR-0012, ADR-0016, ADR-0018, ADR-0020, ADR-0021

## Context

The Eval panel had grown into two panels stacked in one: a gammonNet cube block and a race block
computing the same `gammonnet.Decide` on the same probabilities, in two formattings; tables
unmounted whenever no result was there yet, collapsing the panel at every board edit; and money
equities shown under an "exact" badge at a match score. What made showing the pre-roll vector
always affordable is measured: it costs +36 % of a checker search at 2-ply (1.80 s vs 5.01 s),
and only on a non-race with dice set — every other case already computes it.

## Decision

**A quantity is either a fact of the position or part of a decision, and that determines where
it is shown — always, for every position.** This governs the Eval panel and the part of the
Analysis panel that renders the same quantities.

1. **Facts are per player.** The pre-roll probability vector (win/gammon/backgammon per side)
   and the cubeless equity in the position's Referential are facts, like pip count, EPC,
   wastage, mean rolls and their dispersion. They live in the facts table; the race facts appear
   when the position is a race. Their axis is ADR-0018 rules 1–2; their layout is ADR-0021.
2. **Decisions are per option, and the board asks exactly one question.** Dice on the board →
   the ranked checker plays. No dice → the cube actions and their verdict. Nothing else: a race
   with dice set shows no pre-roll cube verdict.
3. **Structure follows the position, never the state of the calculation.** Tables are never
   unmounted; cells are empty until a value lands and are replaced in place. Across the 0-ply →
   display-depth escalation only the depth label changes. A stale value is never shown dimmed —
   the gesture that invalidates it is the gesture that triggers the recomputation.
4. **Exact wins on what it can answer, and only that.** In the money Referential ADR-0012 holds
   unchanged. At a match score the exact regime keeps the win probability (referential-free);
   equities and verdict come from the evaluated regime, and the badge names both sources.
   `evaluateRaceRegime` short-circuits on "exact **and** money".
5. **The rendering is shared with the Analysis panel, the content rule is not.** Both panels
   mount the same facts table. Eval feeds it the live Evaluation, race facts included; Analysis
   feeds it the stored Analysis — present for a cube record, absent for a checker record (no
   position-level vector), never with race facts. Analysis answers "what did the engine that
   analysed this game say", not "what is this position worth".

**No engine is mixed in one view:** a gammonNet-computed vector is never grafted beside an
imported XG/gnuBG record.

## Consequences

- Only the candidate list scrolls (sticky header); the panel is `overflow: hidden`, so facts,
  verdict and badge are never scrolled away and the no-scroll property no longer depends on the
  panel's height. Default panel sizes are 250 px (bottom) / 420 px (side).
- Layout follows the panel's own width (flex-wrap), never `PANEL_SIDE` vs `PANEL_BOTTOM`.
- The estimated regime's download hint lives in the badge tooltip; the badge links to the
  bearoff settings. The regime stays named on screen (ADR-0012).
- On a race with dice set the pre-roll figure sits beside post-move ones; it is labelled
  pre-roll and set apart (ADR-0018 rule 3), since the gap is the luck of the roll (ADR-0010).
- Rejected: keeping two blocks and raising the default panel height (costs board space for
  everyone, and still scrolls with up to 50 candidates). Rejected: the vector in the decision
  block, XG/gnuBG style (hides the exact two-sided number as soon as dice are set). Rejected:
  the vector wherever it is not redundant (the same quantity would change place). Rejected: a
  "pre-roll" verdict beside the plays (answers a question the board is not asking). Rejected:
  live facts in Analysis (engine mixing).

## Guard

`frontend/tests/e2e/eval-panel-no-scroll.spec.js`.
