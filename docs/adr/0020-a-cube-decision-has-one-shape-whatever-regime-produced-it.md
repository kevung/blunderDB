# A cube Decision has one shape, whatever regime produced it

## Status

accepted — decided 2026-08-31, the same day as ADR-0018 and ADR-0019, and after ADR-0018's result
was looked at. **Completes** ADR-0018 rule 5; amends and supersedes nothing. No engine, no regime,
no stored row and no schema move.

**Lands on top of ADR-0019** (*a displayed match equity is normalised*), merged a few hours
earlier. That decision is about the *scale* the equities are printed in; this one is about the
*shape* they are printed in — orthogonal, and it takes ADR-0019's numbers exactly as they arrive.
One overlap, and it removes work rather than adding it: ADR-0019 restored "too good" by teaching
`cubeActionLabel` to spell it out, so the four-way verdict is no longer lost by the time it reaches
the panel. What ADR-0019 could not fix from where it stood is that the label is a **string in
English** — it had to widen the verdict cell to fit "Too good to double, take" — and that is what
rule 3 below addresses.

ADR-0018 closed with a warning that has to be answered before anything else here is read:

> *This is the second axis decision on the same panel in two days, and the last one that is
> cheap. […] A third reversal would cost the panel its credibility; this one is recorded so the
> next reader argues with the reasons rather than rediscovering them.*

This is not that third reversal, and the claim is checkable rule by rule:

- **ADR-0018 rule 2** (the pre-roll vector takes the axis of the list it heads) — untouched. This
  decision never looks at the vector.
- **ADR-0018 rule 5** (one idiom for all three tables) — *completed*, not reversed. Its scope named
  three components; the race cube verdict is not a component but an inline `<table
  class="decision-race">` in `EPCPanel.svelte:357-398`. It fell through the naming, and that gap is
  exactly where the complaint came in.
- **ADR-0018 rule 4** (provenance collapses into *a single strip*) — applied as written. The
  implementation made it a column inside the content row (`.badges-col`, `margin-left: auto`),
  which is what manufactures the band of white this decision removes — the very failure mode
  ADR-0018 rule «the layout never manufactures a void» was written against.
- **ADR-0018 rule 6** (Défi has three zones) — extended to a leak it did not foresee, never
  contradicted.
- **ADR-0017 rule 2** (the board asks exactly one question) — unchanged, and leaned on below.

What is genuinely new, and what neither ADR ever decided: **the Decision block itself was rendered
in two different shapes depending on which regime produced it.** ADR-0017 partitioned facts from
decision; ADR-0018 chose the facts' axis. Neither ever looked at the decision splitting in two
behind them.

## Context

For a cube position the panel branches on `showRaceDecision` / `showGenericCube`
(`EPCPanel.svelte:297-298`) into two renderings of the same domain object:

| | race — `.decision-race` (`EPCPanel.svelte:357`) | non-race — `CubeVerdictTable` (`EPCPanel.svelte:409`) |
| --- | --- | --- |
| axis | options in **columns**, one row | options in **rows**, three columns |
| distance to best | `gap()` = `v − bestEq`, parenthesised sub-line | «error» column = `equity − best` |
| verdict | a separate chip, translated (`epc.race.verdicts.*`) | a table row, **raw untranslated string** |
| redouble | never — static labels | `cubeValue >= 1 ? noRedouble : noDouble` |
| cubeless | a column — *and* already a column of the facts table | absent (moved out by ADR-0017) |

None of these differences is a decision anybody took. Each is an accretion, and three of them are
plainly defects:

**The two «distances» are the same number.** `gap()` in `EPCPanel.svelte:239` computes
`v − max(ND, min(DT, DP))`; `domaineval.go:303-317` computes exactly that and calls it an error.
One quantity, two names, two typographies.

**The live verdict is shown in English.** `cubeActionLabel` returns `"Double, Take"` — since
ADR-0019, `"Too good to double, take"` as well — and `CubeVerdictTable` renders
`cubeAnalysis.bestCubeAction` raw. That is right for an *imported* record: the string is the
analysing engine's own words, sometimes already localised, which is why the component carries a
`japanese-text` hack for them. It is wrong for our own live evaluation, whose sibling race path
translates the same verdict into nine languages from a key it has had since #126. ADR-0019 made the
string longer and had to let the cell wrap to fit it; a key costs nothing to widen.

**On a race with the cube at 2, the panel writes «Sans double» where the identical non-race
position writes «Pas de redouble».** `.decision-race` never makes the redouble switch. That is not
a matter of idiom.

**The documentation only ever knew about one of the two.** `doc/source/manuel.rst:921-929`
describes *the* cube decision as giving «l'équité cubeless et les équités money […] sous chaque
décision autre que la meilleure, l'écart entre parenthèses» — the race rendering. The non-race one
appears nowhere. A divergence nobody documented is a divergence nobody chose.

Two further findings, surfaced by asking what the unified block would show in every case:

**Three states are rendered as one promise.** `eval.pending` reads «L'évaluation apparaîtra ici.»
It is shown while computing (true), when the regime cannot answer by construction (`race.Eval.Money`
is nil outside the exact regime — ADR-0009's «the cube verdict is never estimated» — so
`showRaceDecision` is false, `showGenericCube` is false because `data.race` exists, and the panel
falls through to the placeholder), and when the engine *refuses* (`Decide` returns `!ok` beyond the
MET horizon, `evaluateRaceRegime` returns nil, or `EvaluatePosition`'s hard error is swallowed by
`.catch((error) => logger.error(...))` at `EPCPanel.svelte:130`). In the last two the promise is
never kept — and ADR-0017 already owed the third as an unshipped follow-up.

**The panel weighs what the engine says cannot be weighed.** `cube.go:673-679`:

```go
action := NoDouble
if owner != CubeOpponent {
    // A cube the opponent owns cannot be turned by the player on
    // roll: the verdict table presupposes doubling is an option, so
    // outside that precondition there is nothing to weigh.
    action = Verdict(eND, eDT, eDP)
}
```

and its match-score twin (`cube.go:713-721`): *«The Crawford game has no cube in play at all, flat
fact.»* Both still return `EquityDoubleTake` and `EquityDoublePass` for actions that cannot legally
be taken, and both panels render them. In columns this is discreet; as three named rows with an
error column it becomes a legible untruth — the panel telling a user they lost 0.4 by not doubling
a cube they do not hold.

## Decision

**A cube Decision is one object with one shape: three named options in rows, plus the verdict.
Nothing about that shape depends on the regime, the panel, or the engine that produced it.**

1. **Options in rows, canonical order, never sorted** — No Double / Double-Take / Double-Pass, in
   that order, always. This applies ADR-0018 rule 2's principle (*a quantity that has a list of
   options to be read against takes that list's axis*) to the decision itself, and then stops: the
   options are **named**, so unlike a ranked play list their order carries no information — ADR-0018
   rule 6's *«the order IS the answer»* is true of `CandidateMovesTable` and false here. Sorting
   would also permute the rows under the eye across the 0-ply → display-depth escalation
   (`EPCPanel.svelte:132-140`), precisely on the close decisions the user is studying, breaking
   ADR-0017 rule 3's *«only the depth label changes»*.
2. **The best option is marked by weight and colour, never by position**, and its error cell is
   **empty, not `+0.000`** — the same rule ADR-0018 rule 3 gives the Baseline band.
3. **The verdict is a typed key on the live path, a string on the stored path, in the same cell.**
   The live wire payload gains the four-way key (`no_double`, `double_take`, `double_pass`,
   `too_good`) — the same four `race.Money.Verdict` already carries, so the panel's two halves stop
   speaking different vocabularies. `domain.DoublingCubeAnalysis.BestCubeAction` does **not** change — it is a storage
   field written by importers, and the batch analysis job that shares `gammonnet.EvaluatePosition`
   keeps writing exactly what it writes today. The key therefore lives on `gammonnet.EvalResult` and
   `GammonNetEvalResult`, never on a `domain` type, so no schema question is opened.
4. **That cell is the single place the block's state is named.** It carries the verdict, or *no
   decision* (a regime not entitled to one), or the refusal, and is empty **only** while a
   computation is genuinely in flight. The refusal must therefore reach the frontend as **data**, in
   the same payload as rule 3 — ADR-0017's owed follow-up, done here because it is the same struct,
   the same serialisation and the same test.
5. **Where doubling is not an option, there is no error to make.** With the cube on the opponent's
   side, or during the Crawford game, all three error cells are empty and the verdict cell names the
   state. The equities stay: they inform, they no longer advise. This is the engine's own comment
   («outside that precondition there is nothing to weigh») finally reaching the screen.
6. **The adaptation between the two data shapes lives in the frontend**, as one derived
   `cubeDecision`, extracted as a pure module of its own — next to
   `positionFacts.js`, never inside it: CONTEXT.md's whole partition is that a Decision is not a
   fact, and a glossary that the file layout contradicts is a glossary nobody believes. `race.Money` keeps its
   regime and its deliberately money-referential exact case (ADR-0017 rule 4); folding it into
   `DoublingCubeAnalysis`, a type with neither regime nor Referential, would push a display
   distinction into storage or lose it. The frontend is already where the exact/evaluated merge
   lives, by an explicit earlier decision — `gammonnet_eval.go` states *«this function itself does
   not know how the two are combined, that merge lives in the frontend's displayRace»*. One place
   composes this object, not two.
7. **Défi masks structurally, emphasis included.** The three rows stay; equities, errors and verdict
   become `···`, **and the best-row emphasis is suppressed** — since rule 2 makes it the verdict's
   only other carrier, leaving it on would let a user solve the exercise by looking for the bold
   line. The opaque 180 px stand-in goes: its justification (*«CubeVerdictTable is a foreign
   component with its own scoped CSS»*, `EPCPanel.svelte:401-405`) stops being true when the
   component is ours.
8. **The strip is a strip.** Regime badge, depth, engine link and the Défi toggle move to their own
   full-width line above the content row, which then holds only the facts table and the decision.
   `margin-left: auto` goes with them, and with it the void: the special case its comment defends
   (dice on a non-race, where `.top-row` holds nothing *but* the badges) is a strip in disguise.

**Analysis reports, it does not correct.** Rules 1, 2, 5 and 8's idiom apply to both panels — one
rendering, ADR-0017 rule 5 — but where a stored record declares a best action whose error is not
zero, the record's own figures are shown. Blanking is a rule of the path where *we* compute
`equity − best`. Analysis keeps its per-record depth/engine footer, its multi-engine repetition, and
its played-action highlight, which stays a **background** so that *played* and *best* remain two
orthogonal channels on one row.

## Considered options

- **Repaint only: keep two blocks, unify their fonts and rules.** Cheapest, and it addresses the
  idiom collision. Rejected: it leaves one object in two axes, which is the complaint.
- **Converge on the race axis** (options in columns, everywhere including Analysis). One row is
  shorter. Rejected: a four-way verdict, an error per option and a played/best highlight all want a
  row apiece, and Analysis stacks several engines — columns do not survive any of that.
- **Sort the three options by equity, best first.** Consistent with `CandidateMovesTable`. Rejected
  under rule 1: named options, and rows that would permute mid-escalation.
- **Adapt in Go: have `evaluateRaceRegime` also return a `DoublingCubeAnalysis`.** One type reaches
  the panel. Rejected under rule 6 — it puts regime and Referential into a type that has neither, and
  creates a second place where this object is composed.
- **Add the verdict key to `domain.DoublingCubeAnalysis`.** Simpler on the frontend. Rejected: it is
  a stored type, so it drags in `DatabaseVersion`, a migration on two backends and the storage
  contract suite, for a display concern.
- **Localise the existing `bestCubeAction` string in the frontend** instead of carrying a key.
  `normalizeCubeAction` could already reduce it, and since ADR-0019 the string does carry all four
  verdicts. Rejected: it makes the panel parse English prose to find out what its own engine
  decided, and it breaks the moment the wording changes — as it just did.
- **Leave the dead-cube case alone** (rule 5). It is pre-existing, identical on both paths, and
  deliberately copied into `race.Money`. Rejected because the row form makes it legible: an error
  column on an impossible action states something false.
- **Keep the badges in the content row and tune the gap.** Rejected: ADR-0018 already ruled on this
  — *«the void was a rule producing it, not a size to be tuned»*.

## Consequences

- **The panel stops changing shape between a race and a non-race cube position.** The badge names
  the regime, as ADR-0012 requires; nothing else moves.
- **`.decision-race`, the decision chip and `.eq-gap` are deleted**, and `CubeVerdictTable`'s
  `width: 28%` with them — inert in this panel (its cells are `white-space: nowrap`, and a table
  never renders below its min-content width) but sized for `AnalysisPanel`'s wide parent and
  misleading wherever it is read.
- **The cubeless equity leaves the decision block entirely.** It was in `.decision-race` *and*
  already a facts-table column for the same position; ADR-0017 rule 1 makes it a fact, and this
  removes the last copy that said otherwise.
- **One vocabulary survives.** `analysis.*` keeps the option labels (it already makes the
  double/redouble switch); the four-way verdict keys keep the verdict. `epc.race.noDouble`,
  `doubleTake`, `doublePass`, `cubeless` and `cubeStates.*` become dead and are **deleted across all
  nine locales** — `i18nOrphanKeys.sync.test.js` fails until they are, and its `DYNAMIC_PREFIXES`
  list (line 39) names two of them, so the test file changes with them.
- **The cube-state tooltip goes.** «videau centré / possédé / adverse» repeats the board — the cube
  is on it, with its value — and is now also carried by the redouble labels and by rule 5's named
  state.
- **A race with the cube at 2 finally says «redouble».**
- **The strip costs ~18 px of height that the badges cost nothing today.** Against ADR-0017's budget
  (a cube position ~140 px, default panel 250 px) that is affordable, and rule 4 of ADR-0018 keeps
  it at the top: the badge qualifies the numbers below it.
- **The no-scroll Playwright assertion, owed since ADR-0017 and re-owed by ADR-0018, ships here.**
  Not out of conscience: rule 8 changes the height budget, and a property nobody measures after
  three layout changes is not a property.
- **A Défi non-leak test ships too**, and is the non-negotiable one. This panel has leaked twice —
  the gammonNet cube table visible during Défi (ADR-0017's opening context) and `CandidateMovesTable`
  mounted with no mask at all (ADR-0018 rule 6). Twice is a pattern.
- **`cubeDecision` is testable without a DOM**, like `moverFactsToSides` before it: exact Money,
  evaluated Money, live `DoublingCubeAnalysis`, estimated-without-decision, refusal.
- **"No decision" needs one more input than it looks like it does.** The fast race path
  (`updateEPC`) fills the estimated block well before gammonNet answers, so a naive reading would
  flash *no decision* — a settled state — at a position still being computed, which is rule 4's own
  lie in the other direction. `cubeDecision` therefore takes whether an evaluation has come back for
  **this** position, and only then is an estimated race allowed to say it has no verdict.
- **Nothing in `pkg/` changes beyond one added field on `gammonnet.EvalResult`** — no engine, no
  regime, no stored row, no wire type the daemon serves, and `DatabaseVersion` does not move.
