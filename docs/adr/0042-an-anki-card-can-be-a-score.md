# An Anki card can be a score

Status: accepted.
See also: ADR-0025, ADR-0026, ADR-0040

## Context

The score card of ADR-0040 rule 4 — take points and gammon values of a score — is the one
training content that is retained rather than calculated; drilling it under the clock does not
make it stick over weeks, which is what FSRS is for. An Anki card was a position
(`AnkiCard.PositionID`) and a deck came from a collection or a search; a score is neither.

## Decision

1. **A card has a kind and a key**: `position` keyed by the position id, or `score` keyed by
   the unordered score (« 3:5 »). The review view renders a score card with the same component
   the Training tab uses — one card, two hosts.
2. **A deck may have `scores` as its source.** The user creates it (« Nouveau paquet › Fiches
   de score ») and the application fills it with the 36 unordered scores of 2–9 away; the user
   never enters a score. It is never created by default. It can be regenerated when the
   reference tables change.
3. **In review, a score card is the card of ADR-0025**: masked, revealed as one block, graded
   in four degrees. Anki reviews and Training sessions do not see each other's logs.
4. **Scheduling replaces drawing**: the deck presents the scores that are due; the Training
   Scores exercise draws at random.

## Consequences

- Card rows carry `kind` and `key` on both backends; every pre-existing card is `position`
  with its id as key.
- The reference tables (`takePoint*`, `gammonValue*`) live in one shared module read by all
  three clients (see ADR-0040).
- Rejected: an automatic deck — 36 cards due on day one is a debt nobody contracted (rule 2).
- Rejected: one card per face or per number — 400 fragments of one decision.
- Rejected: pseudo-positions carrying a score — a lie in the schema to avoid a column.
- Rejected: a scheduler inside Training — a second FSRS.

## Guard

`frontend/src/__tests__/scoreCard.test.js`,
`frontend/src/__tests__/ankiService.scoreCard.test.js`,
`frontend/src/__tests__/AnkiPanel.scoreCard.test.js`,
`pkg/blunderdb/database/migration_test.go`.
