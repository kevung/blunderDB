# A cube Decision has one shape, whatever regime produced it

Status: accepted.
See also: ADR-0017, ADR-0018, ADR-0019

## Context

The cube Decision was rendered in two shapes: the race regime put the options in columns with a
parenthesised "gap" and a translated verdict chip, the non-race path in rows with an "error"
column and the engine's English verdict string. The gap and the error are the same number; the
race path never said "redouble"; three states (computing, no decision by construction, engine
refusal) shared one "evaluation will appear here" promise; and both showed an error for doubling
a cube the player cannot turn (owned by the opponent, Crawford game) — the engine itself says
there is nothing to weigh there.

## Decision

**A cube Decision is one object with one shape: three named options in rows, plus the verdict.
Nothing about that shape depends on the regime, the panel, or the engine that produced it.**

1. **Options in rows, canonical order, never sorted** — No Double / Double-Take / Double-Pass.
   The options are named, so their order carries no information, and sorting would permute rows
   across the 0-ply → display-depth escalation (ADR-0017 rule 3).
2. **The best option is marked by weight and colour, never by position**, and its error cell is
   **empty, not `+0.000`**.
3. **The verdict is a typed key on the live path, a string on the stored path, in the same
   cell.** The live payload (`gammonnet.EvalResult`, `GammonNetEvalResult`) carries the four-way
   key `no_double` / `double_take` / `double_pass` / `too_good` — `race.Money.Verdict`'s
   vocabulary — translated by the frontend. `domain.DoublingCubeAnalysis.BestCubeAction` stays the
   importers' stored string.
4. **The verdict cell is the single place the block's state is named:** the verdict, *no
   decision* (a regime not entitled to one — ADR-0009), or the refusal; empty **only** while a
   computation is genuinely in flight. The refusal reaches the frontend as data in the same
   payload, never as a swallowed rejected promise.
5. **Where doubling is not an option, there is no error to make.** Cube on the opponent's side or
   Crawford game: the three error cells are empty, the verdict cell names the state, the
   equities stay.
6. **The adaptation between the two data shapes lives in the frontend**, as one derived
   `cubeDecision` in its own pure module beside `positionFacts.js` — a Decision is not a fact.
   `race.Money` keeps its regime and its money-referential exact case (ADR-0017 rule 4).
7. **Défi masks structurally, emphasis included.** The three rows stay; equities, errors and
   verdict become `···`, and the best-row emphasis is suppressed, since it would give the answer
   away.
8. **The strip is a strip.** Regime badge, depth, engine link and the Défi toggle sit on their
   own full-width line above the content row, which holds only the facts and the decision.
   No `margin-left: auto`.

**Analysis reports, it does not correct.** Rules 1, 2, 5 and the idiom apply to both panels
(ADR-0017 rule 5), but a stored record's own figures are shown even where its declared best
action has a non-zero error. Analysis keeps its per-record depth/engine footer, its multi-engine
repetition, and its played-action highlight as a background, so *played* and *best* stay two
orthogonal channels.

## Consequences

- The panel no longer changes shape between race and non-race cube positions; the badge names
  the regime (ADR-0012).
- The cubeless equity is a fact only (ADR-0017 rule 1), never in the decision block.
- Option labels come from `analysis.*` (with the double/redouble switch); verdicts from the
  four-way keys. The strip costs ~18 px of height.
- `cubeDecision` needs to know whether an evaluation has come back for **this** position, or the
  fast race path would flash *no decision* while gammonNet is still computing.
- Nothing in `pkg/` changes beyond one field on `gammonnet.EvalResult`; no schema move.
- Rejected: repaint only (one object stays in two axes). Rejected: options in columns everywhere
  (a verdict, per-option error, played/best highlight and multi-engine Analysis each want a row).
  Rejected: sorting by equity (rule 1). Rejected: returning a `DoublingCubeAnalysis` from Go (puts
  regime and Referential into a type with neither; a second composition point). Rejected: the key
  on `domain.DoublingCubeAnalysis` (stored type — drags in `DatabaseVersion` and a migration for a
  display concern). Rejected: parsing the English verdict string in the frontend (breaks when the
  wording changes). Rejected: tuning the badge gap in the content row (a void is a rule, not a
  size — ADR-0021 rule 4).

## Guard

`frontend/src/__tests__/cubeDecision.test.js`, `CubeVerdictTable.rules.test.js`,
`CubeVerdictTable.challengeMask.test.js` (under `frontend/src/__tests__/`);
`frontend/tests/e2e/eval-panel-no-scroll.spec.js`; `i18nOrphanKeys.sync.test.js`.
