# Training is a tab of exercises: it drills what is calculated, Anki keeps what is retained

Status: accepted.
See also: ADR-0025, ADR-0041, ADR-0042

## Context

Anki, the Eval panel's Défi mode and training exercises all look alike to the user — show,
think, reveal, next — but differ in what the gesture is for:

| | what is asked | who judges | question source | paced by |
|---|---|---|---|---|
| Anki | a judgement, or a fact to retain | the user, four grades | the library | FSRS, over days |
| Défi | a calculation | the user, no grade | the board as it is | nothing |
| Training | a calculation, or a decision | the app, or the user ticking faults | generated, or the library | the clock, within a session |

**Anki keeps what is retained; Training drills what is calculated.** Controls belong in the
application's panels, not in a strip above the board.

## Decision

1. **Training is a tab of the panel**, between Eval and Anki (calculate / retain), visible by
   default, opened by `Ctrl+J`, by a toolbar button right after « Position aléatoire », and by
   `train` (bare: opens the tab; `train <exercise>`: opens it and starts with the remembered
   seed source). Every session gesture — start, reveal, validate, next, finish, leave — is in
   the tab; the board shows the question (and for Pips the answer) but hosts no control.
2. **An exercise is four properties, and declares nothing else**: *source* (seed pool, board,
   or library — ADR-0041); *numbers* (label, truth, tolerance each); *surface* (the board, or
   nothing); *answer mode* — **entered** (typed, graded within tolerance, signed deviation
   recorded), **declared** (revealed, all right by default, the user ticks the wrong ones by
   click or Tab + Space), or **chosen** (one of a few options, graded exactly). The mode
   belongs to the exercise, never to the session: what is estimated is entered, what is
   counted or recalled is declared.
3. **Five exercises.**

   | exercise | source | numbers | surface | mode |
   |---|---|---|---|---|
   | Pips | pool, board, library | pip count of both sides (the overlay is hidden until revealed) | board | declared |
   | Bearoff | pool, board, library | EPC of both sides, tolerance 0.5 | board | entered |
   | Scores | pool: 36 unordered scores 2–9 away | the score card (rule 4) | none | declared |
   | Evaluation | pool, board, library; any position, money play | win % (entered), cube action (chosen); exact EPC shown after, when one exists | board | entered + chosen |
   | Decision | library only, analysed positions | the move played on the board (`quizPlay.js`) or the cube action, judged against the stored analysis; session PR | board | chosen |

   Without a library, or with no analysed position, Decision refuses by name.
4. **A score card is one unordered score with two faces**, *you* and *the opponent*, each up
   to seven rows: take point at cube 2 and at cube 4 (long race, last roll), gammon value at
   cube 1 (**gv1**), 2 and 4. Each face shows only the cells the reference tables define
   (gv2: gammon winner ≥ 3 away; gv4: ≥ 5; tp4: both ≥ 3) — three to fourteen numbers, one
   column when the score is level. The journal counts by cell type, not by face.
5. **The clock is seen, may bind, and a session has no length.** A counter runs from display
   to « Révéler » / « Valider » (ticking faults is not timed). A per-question limit is chosen at
   launch: none (default), 15, 30 or 60 s; at the limit the question reveals itself and counts
   **out of time** — every number wrong, no deviation added. « Terminer » records the session,
   « Quitter » discards it.
6. **The journal is two library tables**, `training_session` (exercise, seed source, date,
   numbers asked, faults, mean absolute deviation, median time, PR for Decision) and
   `training_item` (session, number type — `pips.bottom`, `epc.top`, `tp4.last`, `gv2`,
   `cube`… —, wrong or not, signed deviation; for Decision, the position id). No cap. At
   rest the tab shows the launcher and, per exercise, one summary line (sessions, fault rate,
   mean deviation, median time, trend over the last ten) unfolding to per-number detail. For
   Decision, « revoir les ratées » makes the browsed list the positions failed in the last
   session or last *n* sessions. Not in the Stats panel; no chart.
7. **Défi stays untouched**, outside any exercise; Training neither drives nor needs it.

## Consequences

- The reference tables (`takePoint*`, `gammonValue*`) are one shared module read by the
  reference modals, the Scores exercise and the Anki score card (ADR-0042).
- `train` aliases `tp`, `takepoint` → `scores`, `epc` → `bearoff`, `quiz` → `decision`.
- The CLI and `serve` expose no training.
- The corrected take point (combining both faces' gammon values) would be a sixth exercise,
  entered over the Scores source — the test of rule 2.
- Rejected: a training bar above the board — controls live in panels.
- Rejected: Training as generated Anki decks — a generated question has no identity or
  schedule, and Anki grades a memory, not a calculation.
- Rejected: reveal-only everywhere — loses the signed deviation of estimated quantities.
- Rejected: one binary verdict per question, or Anki's four grades — loses which cell fails.
- Rejected: number keys to tick faults — do not scale to fourteen cells.
- Rejected: a session length chosen at launch — « Terminer » suffices.
- Rejected: a JSON metadata key for the journal, or a Stats tab — rule 6.
- Rejected: forcing seven cells with « 0 »/« — » — a convention, not a calculation.

## Guard

`frontend/src/__tests__/trainingTab.test.js`, `frontend/src/__tests__/trainingTabService.test.js`,
`frontend/src/__tests__/scoreCard.test.js`, `frontend/src/__tests__/TabbedPanel.a11y.test.js`.
