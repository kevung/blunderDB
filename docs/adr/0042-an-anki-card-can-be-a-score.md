# An Anki card can be a score

## Status

accepted — decided 2026-09-07, with 0040. Extends 0025 (what a review card asks) and
0026 (how a finite deck is paced): a deck of scores is finite by construction and is paced
like any other. Changes the card schema, which is why it has its own record.

## Context

The score card of 0040 rule 4 — the take points and gammon values of a score, both
faces — is the one training content that is *retained* rather than calculated. Drilling it
under the clock (the Scores exercise) measures how fast one recalls it; it does nothing to
make it stick over weeks. That is what FSRS is for, and FSRS lives in Anki.

An Anki card, as stored, is a position (`AnkiCard.PositionID`), and a deck is built from a
collection or a search (`AnkiDeck.SourceType`). A score is neither.

## Decision

1. **A card has a kind and a key.** `position` with the position's id, as today;
   `score` with the unordered score as key (« 3:5 »). The review card carries either a
   Position or a score; the review view renders the score card with the same component
   the Training tab uses — one card, two hosts.

2. **A deck may have `scores` as its source.** The user creates it as they create the
   others — « Nouveau paquet › Fiches de score » — and the application fills it with the
   36 unordered scores of 2–9 away. The user never enters a score; the one choice at
   that click is to want the deck or not. It is not created by default: a deck that
   appears on its own with 36 cards due on day one is a review debt nobody contracted.
   It can be regenerated when the reference tables change.

3. **In review, the card is the card of 0025.** Masked, revealed as one block on one
   gesture, graded in Anki's four degrees. No tick-the-faults here: Anki schedules a
   memory, it does not measure a calculation, and its review log stays what it is. The
   Training journal does not see Anki reviews, and Anki's statistics do not see Training
   sessions.

4. **Scheduling replaces drawing.** The Scores exercise draws five scores at random under
   the clock; the deck presents the scores that are *due*. Same 36 cards, one picked by
   chance, the other by the user's memory.

## Considered options

- **Create the deck automatically** — rejected under rule 2.
- **One card per face, or per number** — rejected: 36 cards each readable at a glance
  beat 400 that cut a decision into pieces the user reads together.
- **Pseudo-positions carrying a score** — rejected: a lie in the schema to avoid a column.
- **A scheduler inside Training** — rejected: a second FSRS.

## Consequences

- A migration on both backends: two columns on the card (`kind`, `key`), a new deck source
  type; every existing card is `position` with its id as key, so nothing old changes
  meaning.
- The reference tables (`takePoint*`, `gammonValue*`) are read by a third client; they
  move to one shared module, as 0040 already requires.
- The deck's nine-language surface: one new source type in the deck dialog and the help.
