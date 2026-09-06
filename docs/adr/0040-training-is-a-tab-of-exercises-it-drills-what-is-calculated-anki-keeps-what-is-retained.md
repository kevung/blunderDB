# Training is a tab of exercises: it drills what is calculated, Anki keeps what is retained

## Status

accepted — decided 2026-09-07 in a design interview, before any code. Replaces the
training *bar* that I.17 (#273) and J.4 (#294) shipped; keeps their judge, their session
PR and the move-on-the-board reducer (`quizPlay.js`, merged 2026-09-07). Extends 0025 and
0026 (what an Anki card is, see 0042); leans on 0037 (what "the engine plays a few
plies" may and may not mean, see 0041). Does not touch the Eval panel's Défi mode, which
stays exactly what 0018 rule 6 / 0020 rule 7 made it.

## Context

### Three mechanisms that overlap, and one bar that nobody wanted

blunderDB had three ways to train, grown one at a time:

- **Anki** — spaced repetition (FSRS) of a *judgement* on positions of the library, plus
  a cram mode that draws at random.
- **Défi**, in the Eval panel — a display mode: every edit re-masks the two EPCs and the
  cube decision of *whatever is on the board*, and the user reveals them one by one.
  No grade, no clock, no generated position.
- **`train`** (I.17, J.4) — five timed questions in a *bar* above the board: pip count
  (one side only, the roller's), EPC (drawn from the browsed list, contact positions
  discarded, up to sixty draws), take point (the 2-cube long-race table only), and quiz
  (the move or the cube action, graded against the stored analysis). Typed answers,
  tolerances, a signed error, a session summary in the status bar, and a history of at
  most fifty sessions in a JSON key of the library's metadata — a history **no view
  ever read** (`loadSessions` had no caller outside its tests).

The bar was the problem named first: the application's inputs live in its panels, and a
strip of controls above the board broke that with no other justification than being
next to the position. The typed answer came second: reading a take point table in one's
head and then typing seven numbers is a chore that teaches nothing the reveal-and-check
gesture of Défi does not teach better — except where the *size* of the error is itself
the lesson, which is what the interview had to sort out.

### What separates the three is not the flow

Seen from the user, a training question and an Anki card look alike: show, think,
reveal, next. The resemblance is the trap 0025 already named for Défi. What separates
them is **what the gesture is for**:

| | what is asked | who judges | where the question comes from | paced by |
|---|---|---|---|---|
| Anki | a judgement, or a fact to retain | the user, four grades | the library | FSRS, over days |
| Défi | a calculation | the user, no grade | the board, as it is | nothing |
| Training | a calculation, or a decision | the app, or the user ticking faults | generated, or the library | the clock, within a session |

**Anki keeps what is *retained*. Training drills what is *calculated*.** A take point
table is retained: it belongs to Anki by nature, and to Training only as a timed drill.
An EPC is calculated: it belongs to Training, and Anki has no business scheduling it.
The same content — a score card — may live in both, and that is not a duplicate: one
component, two hosts, two calendars.

## Decision

1. **Training is a tab of the panel, and the bar is removed.** It sits between Eval and
   Anki (the order *calculate / retain*), visible by default, opened by `Ctrl+J`
   (the tab convention: `Ctrl+E` Eval, `Ctrl+K` Anki), by a toolbar button placed right
   after « Position aléatoire » (both are exercise gestures, not management), and by the
   `train` command: bare, it opens the tab; `train <exercise>` opens it and starts with
   the remembered seed source. Every gesture of a session — start, reveal, validate,
   next, finish, leave — is in the tab. The board shows the question and, for pips,
   reveals the answer; it never hosts a control of its own.

2. **An exercise is four properties, and a new exercise declares them — nothing else.**

   - *source* — what makes a question: a seed pool (canonical positions or scores), the
     board as it stands, or a position of the library (0041);
   - *numbers* — the values asked, each with a label, a truth and a tolerance;
   - *surface* — what, outside the tab, shows the question or the answer: the board, or
     nothing;
   - *answer mode* — **entered** (the user types, the app grades within the tolerance
     and records the signed deviation), **declared** (the user reveals, every number is
     right by default, and ticks the ones they got wrong), or **chosen** (one of a few
     options, graded exactly).

   The mode is a property of the exercise, never a per-session switch: what is
   *estimated* is entered, because the size of the error is the lesson (« I overestimate
   positions with gaps »); what is *counted or recalled* is declared, because a pip count
   or a table cell is right or wrong and typing it teaches nothing. The tab knows both
   gestures and shows one at a time.

3. **Five exercises ship first.**

   | exercise | source | numbers | surface | mode |
   |---|---|---|---|---|
   | **Pips** | pool (opening + 6–40 plies), board, library | pip count of *both* sides | board; its pip count overlay is hidden while the question is open, and « Révéler » shows it | declared |
   | **Bearoff** | pool (bear-in shapes + 0–10 plies), board, library | EPC of both sides (two fields, tolerance 0.5) | board | entered |
   | **Scores** | pool: the 36 unordered scores 2–9 away | the score card (rule 4) | none | declared |
   | **Evaluation** | pool (contact-broken shapes + plies), board, library — any position, money play in v1 | win chances in % (entered), the cube action (chosen), and the EPC *shown* after the answer when the position has an exact one | board | entered + chosen |
   | **Decision** | library only — a position must carry an analysis | the checker move played on the board (`quizPlay.js`) or the cube action (three buttons), graded by the engine's judge against the stored analysis; session PR kept | board | chosen |

   « Bearoff », not « EPC »: the EPC is the *number*, the bearoff is the *position*, and
   an exercise is named after what one looks at. « Evaluation », not « long race »: its
   domain is any position since a seed may be a contact position (0041), and the name
   echoes the Eval panel on purpose — it is what that panel shows, asked before it is
   shown.

4. **A score card is the score, with its two faces.** One question is one *unordered*
   score (36 on 2–9 away); the card has two columns — *you* and *the opponent* — and
   seven rows: take point at cube 2 (long race, last roll), at cube 4 (long race, last
   roll), gammon value at cube 1, 2 and 4. Each column renders only the cells the
   reference tables define for that face (gv2 needs the gammon's winner ≥ 3 away, gv4
   ≥ 5, tp4 both sides ≥ 3): from three numbers at 2a–2a to fourteen; one column when
   the score is level. No « n/a » cell to guess, no score skipped for being incomplete —
   both would have removed 2-away, the score one meets most. The two faces are there
   because a cube decision at a score needs both: the corrected take point combines the
   gammon values of both players, and the opponent's take point is what says whether
   one's double passes. The journal counts by **cell type** (« tp4, last roll »), not by
   face: the same cell of the same table seen from either side is one weakness, not two.
   The gammon value at a centred cube is named **gv1**, as the tables name it.

5. **The clock is seen, may bind, and the session has no length.** A counter runs in the
   tab from the moment a question is shown to « Révéler » / « Valider »; ticking faults
   is not timed. A per-question limit is chosen at launch — none by default, else 15,
   30 or 60 s; at the limit the question reveals itself and counts **out of time**:
   every number wrong, no deviation added to the mean (one does not measure an answer
   that was not given). A session runs until « Terminer », which records it, or
   « Quitter », which discards it. Five questions was an implementation convention: a
   pool of scores or generated races is infinite, and the library bounds Decision by
   itself.

6. **The journal is two tables of the library, and the tab reads it at rest.**
   `training_session` (exercise, seed source, date, numbers asked, faults, mean absolute
   deviation where one exists, median time, PR for Decision) and `training_item`
   (session, number type — `pips.bottom`, `epc.top`, `tp4.last`, `gv2`, `cube`, … —,
   wrong or not, signed deviation where one exists). In the library, as the old JSON key
   was, because the journal is about *this* library and travels with the file; in
   tables, because the per-number detail is the whole point (« tp4 last roll: 6 faults
   in 9 ») and a JSON blob that grows by one entry per revealed number is the register
   the old code refused for the wrong reason — space, which a table settles. No cap.
   When no session runs, the tab shows the launcher *and*, per exercise, one line of
   summary — sessions, fault rate, mean deviation, median time, the trend over the last
   ten — and unfolds the per-number detail on click. Not in the Stats panel: Stats is
   about the positions one *played*; this is about the questions one *asked oneself*,
   and a Stats tab that stays empty for whoever does not train would occupy a permanent
   place for nothing. No chart in v1: a trend in figures is enough.

7. **Défi stays, untouched, outside any exercise.** It is the Eval panel's own way of
   hiding what it shows, for a position the user has in front of them, with no clock and
   no grade — a convenience 0025 already declined to imitate. Training does not drive it
   and does not need it: the tab and the Eval panel are never visible together.

## Considered options

- **Keep the bar, move the input into it** — rejected: the bar *was* the objection.
- **Merge Training into Anki as generated decks** — rejected. A card is a position with
  an FSRS state; a generated question has no identity, no schedule and no deck, and
  Anki grades a memory in four degrees, not a calculation in half-pips. The one thing
  that *is* retained — the score card — does go to Anki, as a card (0042).
- **Reveal everything, never type** — rejected for the estimated quantities (EPC, win
  chances): the signed deviation is information no tick produces, and it is the only
  reason to keep a keyboard in the loop.
- **One binary verdict per question** — rejected: on a fourteen-number score card it
  loses exactly what one wants to know, which cell keeps failing. Anki's four degrees —
  rejected: an FSRS scale, not a measure of a calculation.
- **Number keys `1`–`7` to tick faults** — rejected once the score card reached
  fourteen cells; a click on the cell (or Tab + Space) is one gesture for every exercise.
- **A session length chosen at launch** — rejected as one setting too many once
  « Terminer » exists.
- **Extend the JSON metadata key** instead of tables — rejected under rule 6.
- **A Training tab in the Stats panel** — rejected under rule 6.
- **Force seven cells on every score card with « 0 » or « — » as the expected answer** —
  rejected under rule 4: knowing gv4 has no object at 3 away is a convention, not a
  calculation.

## Consequences

- `TrainingBar.svelte` and `trainingMaskStore` go; `trainingService.grade` and the
  tolerances stay for the entered mode; `quizPlay.js`, its store and the board wiring in
  `boardInteractions.js` are rebound to the tab unchanged.
- The `takePoint*`/`gammonValue*` tables become answer sources shared by the reference
  modals, the Scores exercise and the Anki score card — one copy, three readers.
- Two new storage tables and their migration, on both backends; the CLI and `serve`
  gain nothing in v1.
- Nine languages of interface strings and documentation; the `train` help entry is
  rewritten, and the aliases `tp`, `takepoint` → `scores`, `epc` → `bearoff`,
  `quiz` → `decision` keep the fingers' memory intact.
- Decision without a library, or with a library that carries no analysis, refuses by
  name — as the old `train quiz` did.
- The corrected take point (combining both faces' gammon values) is a sixth exercise
  waiting to be declared: an entered number over the Scores source. It is the test of
  rule 2, and it is deliberately not in v1.
