# A review card asks one question and expects one grade

## Status

accepted — decided 2026-09-02. Touches the Anki panel and extracts a presentational view out of
the Analysis panel. Changes no schema, no scale, no engine, and no scheduling: FSRS sees exactly
the ratings it saw before.

Its neighbour is the Eval panel's Défi (challenge) mode — ADR-0018 rule 6 / ADR-0020 rule 7 —
which this decision deliberately does *not* imitate. The relationship is stated in rule 2 below
because the resemblance is the trap: two panels that both hide a result are not two instances of
one mechanism.

## Context

### The panel asked a question it never answered

The Anki panel's review view showed the deck, the card number, the card state, `Position #id`, and
four grading buttons. The answer — what the position's stored analysis says — appeared nowhere.
The user thought of a move, and then graded their own recall against nothing, or left the tab to
find out.

### Leaving the tab showed the wrong answer

`ankiService.showCard()` set `positionStore` directly instead of going through
`positionService.showPosition()`, the single function that calls `LoadAnalysis(position.id)` and
fills `analysisStore`. So during a review, `analysisStore` still held the analysis of whatever
position was last browsed. The Analysis tab was one `Ctrl+L` away (the review key guard at
`keyboardService.js:141` excludes `event.ctrlKey`) and it showed a **different position's**
analysis, silently. Every reading of "the answer is already reachable, so hiding it is theatre"
was, until this decision, a reading of a bug.

### The Eval panel's Défi is a different problem

Défi hides three zones independently — bottom EPC, top EPC, cube decision — and each is revealed
on its own click. That granularity is right there: the three are independent facts, and checking
one's own pip count without being told the cube decision is a real thing to want. A review card is
not that. It poses one question, and FSRS records one number.

The mask also cannot be built the same way. `cubeRows()` masks **in place** — the option labels of
a cube decision are fixed («No double», «Double, take», «Double, pass»), so replacing the equities
with `···` hides everything that matters while keeping the table's shape. A checker analysis has
no such property: its rows *are* the moves, ordered by descending equity, so an in-place mask
would leave the answer on the first line. `EPCPanel` already carries `.moves-masked`, an opaque
block, for exactly this reason.

## Decision

1. **The answer of a card is its stored analysis.** Not a live gammonNet evaluation. A deck is
   built from a collection or a search — in practice, blunders imported from XG or gnuBG — and the
   stored record is both the answer the user is checking and the judgement that flagged the error
   they are reviewing. A live evaluation would cost time on every card and could disagree with the
   record, making "the answer" ambiguous. The Eval panel remains one tab away for whoever wants to
   dig. This makes `showCard` go through `showPosition`, which is a bug fix, not a feature.

2. **The mask is a discipline, not a lock.** The Analysis tab stays reachable during a review, the
   Eval panel too. blunderDB does not police its user: Défi is a checkbox one can uncheck, and
   this is the same posture. Locking the answer away would couple three panels to the review state
   and introduce a rule — "which panel may speak right now" — that exists nowhere in this app.

3. **One block, one gesture, one grade.** The whole answer is revealed at once, by clicking the
   masked block or pressing Space. No independent zones. Partial reveals would produce grades
   whose meaning nobody can reconstruct — "I saw the probabilities but not the verdict, is that
   Hard?" — and FSRS has no field for that nuance. It follows that the masked stand-in is a single
   opaque block for both kinds of decision: two visual vocabularies for one action would be
   learned twice for no gain, and `CandidateMovesTable` needs no `masked` prop at all, because the
   shared view is simply not rendered while the answer is hidden.

4. **Revealing is not required in order to grade.** 1-4 stay live whether the answer is shown or
   not. The grade is the user's judgement of their own recall, and they may be certain. Requiring
   a reveal would add a mandatory gesture to every card, and would make cram mode — one "next"
   button, no answer to judge — incoherent.

5. **What resets the mask is a change of question, not a change of view.** The answer re-hides on
   the next card, on starting a session, and on leaving the review; it does **not** re-hide when
   the user switches tabs and comes back. `TabbedPanel` destroys and remounts its child on every
   tab switch, so the reveal state lives in `ankiStore`, not in component state. Going to look at
   the Eval panel or the comment after discovering one's mistake is the behaviour this feature
   exists to encourage; erasing the answer on return would punish it.

6. **The rendering is shared with the Analysis panel, the panel's state is not.** A presentational
   `AnalysisView` renders "the stored analysis of a position"; sort state, cube/checker tabs, MATCH
   mode and keyboard handling stay in `AnalysisPanel`. The review view uses it with a fixed sort
   and no tabs, keeps the played-move highlight — the move actually played is the blunder being
   reviewed — and keeps click-to-show-a-move-on-the-board, which is the moment of learning. It
   drops interactive sorting, which no card needs.

## Considered options

- **Mask the Analysis tab and the Eval panel for the duration of a review** — rejected under rule
  2. It makes Anki reach into two panels it does not own.
- **Capture `Ctrl+L` so the Analysis tab is unreachable during a review** — same rejection, plus it
  takes a global shortcut hostage.
- **Reuse `AnalysisPanel` whole, with a `masked` prop** — rejected: two instances would put
  `id="analysisPanel"` in the DOM twice, and its `onDestroy` clears `selectedMoveStore` under the
  other instance's feet.
- **A persisted "Défi" checkbox for the Anki panel, mirroring the Eval panel's** — rejected. The
  Eval panel needs one because it has a purpose outside Défi; the review view does not — hiding
  *is* its purpose. A setting saved to config, documented and translated into nine languages, to
  save one keystroke per card, is not a trade this project makes.
- **Filter positions without a stored analysis out of decks** — rejected: it would silently amputate
  existing decks, and a position with a comment and no analysis can still be worth reviewing. Such
  a card shows "no analysis recorded" directly, unmasked: an absent answer is not a hidden one, and
  a mask that reveals nothing is a lie the interface tells.

## Consequences

- `selectedMoveStore` must be cleared when the card changes and when the review is left. Left set,
  it freezes j/k position browsing app-wide — a regression this project has already shipped once
  (see `AnalysisPanel`'s `onDestroy` comment).
- Space is spent. It was free during a review (the guard at `keyboardService.js:141` swallows every
  non-Ctrl key), and it is now the reveal. It is deliberately **not** given a second meaning after
  the reveal, unlike real Anki where Space then grades "Good": a double-tap would enter a false
  grade, and a false grade durably pollutes the schedule.
- `showCard` becomes async and now also loads the position's **comment**, since `showPosition`
  does — on a review card, that comment is often the note the user wrote for themselves.
