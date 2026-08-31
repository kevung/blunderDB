# The panel shows position facts, plus the one decision the board asks

## Status

accepted and applied — decided 2026-08-31, implemented the same day on `feat/eval-panel-redesign`.
Refines how ADR-0009 and ADR-0012 are *displayed*; it changes no engine, no regime and no stored
data. Governs the Eval panel, and the part of the Analysis panel that renders the same
quantities. Two follow-ups noted in *Consequences* were not done in this pass: the dedicated
no-scroll Playwright assertion, and surfacing a refused evaluation as a named state rather than a
swallowed error. Every other consequence below shipped.

**Depended on ADR-0016** (*the referential is a property of the position, and the search honours
it*), proposed the same day — a one-way dependency: this decision never waited on that one, only
one of its columns did, and ADR-0016 landed the same day too (`cd7eea98`). Two points of contact:

- Fact 1 below puts a **cubeless equity** in a table that also carries cubeful equities. Before
  ADR-0016, the search stayed money-only at every score, so those two would have sat in different
  Referentials at a match score — what ADR-0016 calls a table that cannot be read. With ADR-0016
  landed, `CubelessValue(&probs, state)` already returns the figure in the position's own
  Referential (money at money play, normalised equity at a match score — ADR-0019; this ADR said
  2×MWC−1 — matching Cubeful*Equity beside it),
  so the column ships filled in both cases, with no gating needed on this side.
- The **Equity column of the candidate-moves table** is ADR-0016's subject, not this one's. Two
  different columns share the word: here, *cubeless equity* always names the position fact, and
  the moves table's per-play equity is never called that.

Decision rule 4 below applies ADR-0016's principle to a place ADR-0016 does not reach and never
will: the two-sided table, where no search is involved and the stored equities are money by
construction — that half of the dependency does not resolve by ADR-0016 landing, and stays.

## Context

The Eval panel grew by accretion: the Bearoff panel gained an evaluation volet (#125), then a
third race regime (#126), each landing as its own block under the previous one. The result
reads as two panels stacked in one, and the seams are measurable rather than aesthetic:

- **The two blocks are the same computation.** On a race outside the exact domain with no dice,
  `evaluateCube` and `evaluateRaceRegime` both call `gammonnet.Decide` on the same `probs`, the
  same `MatchState`, the same efficiency. Two tables, one calculation, two formattings — and
  two places to read a win probability that is one number.
- **The panel is laid out against its own geometry.** `.tables-container` is
  `flex-direction: column`, so in bottom mode — where the panel spans the whole window — the
  content stacks in ~600 px and leaves two thirds of the width empty while asking for height
  that is not there. `CubeVerdictTable` is mounted there without the row flex `AnalysisPanel`
  gives it, so its three sub-tables stack too: three times the necessary height.
- **The training mode leaks.** Défi masks the `bottom`/`top`/`race` zones while the gammonNet
  cube table above them shows win chances and equities in clear.
- **The structure depends on the calculation.** With no result yet, the tables are unmounted in
  favour of a one-line placeholder: every board edit collapses the panel from ~100 px to ~20 px
  and back.
- **The exact regime is money-referential.** `race.Evaluate` builds `Money` via `MoneyFromEntry`
  without ever reading `pos.Score`. At a match score the panel therefore shows *money* equities
  under an "exact" badge, in the cells where the engine would put *match* equities.

One number decided the shape of the answer, and it was measured on this build rather than
assumed (opening position, 3-1, k=12; the first draft of this decision guessed it wrong by a
factor of three, in the direction that would have justified the lazier design):

| depth | `Plays` (a checker decision) | `Probs` (a pre-roll vector) | extra |
| ----- | ---------------------------- | --------------------------- | ----- |
| 0-ply | 7 ms, 12 evals               | 1 ms, 1 eval                | −87 % |
| 2-ply | 5.01 s, 13 439 evals         | 1.80 s, 4 762 evals         | +36 % |

A pre-roll probability vector is not a second search's worth of work. And it is already paid
for in every case but one: free in the exact regime (a lookup), already computed by
`evaluateRaceRegime` outside it, already computed by the cube evaluation when no dice are set.
The only position that pays the +36 % is a **non-race with dice on the board**, at display
depth, inside a background search that is already debounced by 500 ms and cancellable.

## Decision

**A quantity is either a fact of the position or part of a decision, and that determines where
it is shown — always, for every position.**

1. **Facts are per player.** The pre-roll probability vector (win/gammon/backgammon for each
   side) and the cubeless equity, in the position's Referential, are facts — exactly as pip
   count, EPC, wastage, mean rolls and their dispersion are. They live in one table whose rows are `bottom` / `top` / `Δ`. The
   first columns are always present; **the race columns appear when the position is a race**,
   and that appearance *is* the transition between the two former panels — the table does not
   move, five columns arrive.
2. **Decisions are per option, and the board asks exactly one question.** Dice on the board →
   the ranked checker plays. No dice → the cube actions and their verdict. Nothing else is
   ever shown: in particular a race with dice set no longer displays a pre-roll cube verdict,
   which answered a question the board was not asking.
3. **Structure follows the position, never the state of the calculation.** Tables are never
   unmounted; cells are empty until a value lands and are replaced in place. Across the
   0-ply → display-depth escalation only the depth label changes. A stale value is never shown
   dimmed — the gesture that invalidates it is the gesture that triggered the recomputation
   (the same reason the selected move's arrow is cleared on entering the escalation, not on
   its result).
4. **Exact wins on what it can answer, and only that.** ADR-0012's rule is unchanged in the
   money Referential. At a match score, exact keeps the **win probability** — which is
   referential-independent — and the equities and verdict come from the **evaluated** regime,
   which ADR-0012 explicitly makes available at a score. The badge names both sources. The
   short-circuit in `evaluateRaceRegime` therefore becomes "exact **and** money", not "exact".
5. **The rendering is shared with the Analysis panel, the content rule is not.** Both panels
   mount the same facts table. In Eval it is fed by the live Evaluation and carries the race
   columns; in Analysis it is fed by the stored Analysis, so it is present for a cube record
   and absent for a checker record — the record has no position-level vector — and it never
   carries race columns. Analysis answers *"what did the engine that analysed this game say"*,
   not *"what is this position worth"*.

**No engine is ever mixed in one view.** Analysis shows imported records (XG, gnuBG); a
gammonNet-computed vector will not be grafted beside them. ADR-0013 says as much about the
data; the same restraint applies on screen.

## Considered options

- **Keep the two blocks and size the panel around them.** This is what shipped in `ca9e7b65`:
  the default panel height raised from 250 px to 380 px so the stacked tables fit. It
  buys the no-scroll property with 130 px of board, permanently, for every user — and it fails
  anyway, since the candidate count is user-settable up to 50.
- **Put the probability vector in the decision block** (columns per move, rows per cube
  action) and keep the facts table strictly EPC. Zero duplication and zero extra computation,
  and it is what XG and gnuBG do. Rejected once the +36 % was measured: it hides the exact
  two-sided number — the oracle ADR-0012 keeps precisely because a user can check it against a
  book — the moment dice are placed on the board, in the panel that exists to show it.
- **Show the vector where it is not redundant** (facts table when dice are set, decision block
  otherwise). Rejected: the same quantity would change place depending on the position, which
  is the opposite of a smooth transition.
- **Keep the pre-roll cube verdict beside the candidate plays**, labelled "pre-roll". Rejected:
  it spends a table in the only case that scrolls, to answer a question the board is not
  asking.
- **Give Analysis the live facts too.** Rejected — see the engine-mixing rule above.

## Consequences

- **Only the candidate list scrolls.** The panel is `overflow: hidden`; the moves table becomes
  the single scrollable region, with a sticky header. The facts table, the verdict and the
  regime badge are therefore never scrolled out of view, and the no-scroll property stops
  depending on the panel's height. A cube position fits in ~140 px against ~215 px stacked
  today.
- **The panel-size defaults go back to 250/420**, reverting that part of `ca9e7b65`. The
  persistence that commit added is kept — it is independently useful — but the raised defaults were sized for a layout that no
  longer exists.
- **The layout is driven by the panel's own width, never by `PANEL_SIDE` vs `PANEL_BOTTOM`.**
  One `flex-wrap` flow under container queries (~1000 px → one row; 600–1000 px → two;
  < 600 px → stacked), so a side panel dragged wide behaves like a bottom band. Cell padding
  drops from `1px 18px` to `2px 10px`, which is what brings the single-row threshold within
  reach of a modest window.
- **Défi regains a sound definition**: three zones — the `bottom` row, the `top` row, and the
  decision block — and nothing escapes the mask.
- **The download hint stops costing height.** The "estimated" regime's paragraph folds into the
  badge's tooltip and the badge becomes the link to the bearoff settings. ADR-0012's
  requirement that the regime be named on screen is untouched; it is the nag that goes, not the
  name. The gammonNet attribution and the Défi toggle join the same badge column, which frees
  the 80 px gutter `.tables-container` reserved for the pinned toggle.
- **The no-scroll property becomes a test**, not an intention: a Playwright assertion that
  `scrollHeight <= clientHeight` for a cube position, race and non-race, **at the default panel
  size** — not at any size, which would assert the impossible.
- **On a race with dice set, the pre-roll win probability stays and the pre-roll verdict goes.**
  The first is a fact and survives (that is what the +36 % buys); the second is a decision the
  board is not asking for. The facts table will therefore show a pre-roll figure directly beside
  a list of post-move ones, which differ by the value of the roll — the luck of the position
  (ADR-0010). The facts table must be **labelled as pre-roll** and set apart visually, or one
  confusion has simply replaced another.
- `CubeVerdictTable` loses its `left-table`; the quantities move into the shared facts table.
  Both panels change together, by construction.
- **The cubeless-equity column shipped filled at every score, not gated.** By the time this
  landed, ADR-0016 had too (`cd7eea98`, same day): `evaluateCube` now threads the search's own
  `MatchState` into `CubelessValue(&probs, state)`, so the figure is already in the position's
  Referential — money at money play, normalised equity at a match score (ADR-0019 amends the
  2×MWC−1 written here), matching `Cubeful*Equity` beside
  it in the same table. No frontend gating was needed; an earlier draft of this decision had one
  (strip the column to empty whenever a score was present) and it was removed once the dependency
  resolved, rather than shipped as a harmless-looking leftover that would have quietly hidden a
  now-correct number.
- **An engine that refuses is not yet an empty cell, and it should be — not shipped.** With
  ADR-0016 landed, `EvaluatePosition` now returns a hard error for a match state beyond the MET's
  horizon (never a silent fallback to money, which is the point of that decision). But that error
  currently only reaches `internal/gui/gammonnet_eval.go`'s caller as a rejected promise, logged
  and swallowed by the frontend (`.catch((error) => logger.error(...))`) — the panel is left
  showing whatever it already had, stale, rather than a cell that names the refusal. Distinguishing
  "refused" from "still computing" at the badge, as this decision intended, needs the error to
  reach `GammonNetEvalResult` as data rather than a rejected call; left as a follow-up.
- **A defect surfaced while writing this, worth fixing in the same branch.** `CubeVerdictTable`
  renders `cubelessNoDoubleEquity` and `cubelessDoubleEquity`, and `evaluateCube` sets neither.
  Every live cube evaluation in the Eval panel therefore shows two rows of `+0.000` today. The
  facts table replaces those two rows with the single cubeless-equity column above, which removes the
  defect rather than relocating it.
