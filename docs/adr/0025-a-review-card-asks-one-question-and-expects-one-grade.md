# A review card asks one question and expects one grade

Status: accepted.
See also: ADR-0026, ADR-0042

## Context

A review card that shows a position and four grading buttons, but never the answer, makes the
user grade their recall against nothing. A card that sets `positionStore` without going
through `positionService.showPosition()` leaves `analysisStore` holding another position's
analysis, one `Ctrl+L` away. The Eval panel's Défi mode hides three independent facts; a card
poses one question and FSRS records one number, and a checker analysis cannot be masked in
place (its rows are the moves, best first).

## Decision

1. **The answer of a card is its stored analysis**, not a live gammonNet evaluation. It is the
   record that flagged the error under review; a live evaluation costs time and could disagree.
   `showCard` goes through `showPosition` (which also loads the comment).
2. **The mask is a discipline, not a lock.** The Analysis tab and the Eval panel stay
   reachable during a review; blunderDB does not police its user.
3. **One block, one gesture, one grade.** The whole answer is revealed at once, by clicking
   the single opaque masked block or pressing Space. No independent zones; the shared view is
   simply not rendered while hidden.
4. **Revealing is not required to grade.** Keys 1-4 stay live either way.
5. **A change of question resets the mask, a change of view does not.** The answer re-hides
   on the next card, on starting a session and on leaving the review, not on switching tabs
   and back; the reveal state lives in `ankiStore` (`TabbedPanel` remounts its children).
6. **The rendering is shared with the Analysis panel, its state is not.** The presentational
   `AnalysisView` renders a stored analysis; sort, tabs, MATCH mode and keys stay in
   `AnalysisPanel`. The review uses a fixed sort, keeps the played-move highlight and
   click-to-show-on-board.

## Consequences

- `selectedMoveStore` is cleared when the card changes and when the review is left; left set,
  it freezes j/k browsing app-wide.
- Space is the reveal and gets no second meaning after it (a double-tap would enter a false
  grade that durably pollutes the schedule).
- A card whose position has no stored analysis shows "no analysis recorded", unmasked.
- Rejected: masking the Analysis/Eval panels or capturing `Ctrl+L` during a review — Anki would
  reach into panels it does not own (rule 2).
- Rejected: reusing `AnalysisPanel` with a `masked` prop — duplicate DOM id, and its
  `onDestroy` clears `selectedMoveStore` under the other instance.
- Rejected: a persisted Défi checkbox for Anki — hiding is the review's purpose, not an option.
- Rejected: filtering analysis-less positions out of decks — silently amputates decks.

## Guard

`frontend/src/__tests__/ankiService.showCard.test.js`,
`frontend/src/__tests__/keyboardService.ankiReveal.test.js`.
