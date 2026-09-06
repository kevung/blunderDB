# The game phase is derived, never edited, and recomputable

## Status

accepted — implemented in 2.19.0 (`engine/gamephase.go`, `position.game_phase`),
issue #264 (fiche I.8).

## Context

Every statistic blunderDB shows is an average over the whole database. A player
who wants to know whether they lose their points in the race or in contact
cannot ask: there is no column that says which. The fifty-odd search filters
describe the *board* (pip counts, back checkers, occupancy masks); none of them
names a **phase of the game**, and reconstructing one from a conjunction of
filters — `nc` plus a pip range plus an off count — is both wrong at the edges
and unavailable to the statistics, which do not run the search grammar.

The obvious implementation is a column. The two questions it raises are what
the values mean, and who owns them.

The research report
[P5](../recherche/P5-classification-type-de-jeu.md) settles the first
question in a way that constrains the second. Its finding, from reading gnubg's
`ClassifyPosition` and the backgammon literature: **the only classification
boundaries that are sourced and deterministic are gnubg's** — over, race
(the two rearmost checkers have crossed), crashed. Everything a human player
calls a *plan* — holding, backgame, blitz, prime-vs-prime — falls into gnubg's
single `CONTACT` class, and no publication states a threshold between them.
P5's own recommendation is that any unsourced threshold be "un paramètre nommé
et versionné", and that the taxonomy be validated by hand-sampling rather than
presented as a standard.

That is an argument for a *small* first label, not for the ten-way plan
classifier. The plan classifier is a separate, larger chantier (J.1, #291);
the phase is what can be defended today.

## Decision

**`position.game_phase` is a derived label: opening, middlegame, race,
bearoff.** Four values, plus `unknown` for a row nothing has classified yet.

**It is computed from the board alone**, by `engine.ClassifyGamePhase`. Not
from the move number of the game the position came from — a position is
identified by its Zobrist hash and can be reached at move 3 of one match and
move 30 of another, so a label derived from the visit would contradict itself
between two rows that are the same row. Not from the cube, the score or the
side on roll either: the classification is symmetric, and holds whichever
player is on roll.

**Three of the four boundaries are gnubg's, and are sourced.** Contact vs. race
is exactly `domain.Position.MatchesNoContact`, which is the crossing test P5
attributes to `ClassifyPosition`. Bearoff is the point from which every checker
still on the board stands in its own home board.

**The fourth is a convention, and says so.** `engine.OpeningDisplacementMax`
= 4 is the largest number of checkers either side may have moved off its
starting points for the position to still count as the opening — the count a
side reaches after two ordinary rolls, or after one doublet. It is a named
constant with its reasoning in its doc comment, exactly as P5 asks. Nothing
else in the code base compares against 4.

**It is never editable.** No command, no panel, no route sets a phase. There is
no "the classifier is wrong here" affordance, and that is deliberate: an
editable derived column is two sources of truth, and the first user who edits
one makes every later change to the rules unappliable.

**It is recomputable, and recomputing it is a supported gesture.**
`blunderdb repair` reclassifies every position whose stored phase disagrees
with the classifier, and the 2.19.0 migration is that same pass. Change the
rules or the threshold, repair, and every row agrees with the new rule.
Nothing else in the database depends on the old value.

## Consequences

The `ph` search token, the phase column of the statistics (I.10, #266) and the
per-phase PR the plan promises all read one column with an index behind it,
rather than each deriving a phase of its own.

A database upgraded to 2.19.0 carries `unknown` on every row until the
migration's backfill runs — which it does, in the same open. A database whose
`state` cannot be decoded classifies as `unknown` and is written as such rather
than skipped: "we do not know" is a phase, and `blunderdb verify` can count it.

The label is a **convention, not a standard**, and the documentation says so
where it is user-visible. P5 found no published inter-annotator agreement for
any backgammon type classification; claiming otherwise would be inventing a
provenance.

A future change to the rules is cheap *because* nothing may edit the column.
That is the whole trade: the user cannot correct a position, and in exchange
the maintainer can correct every position at once.
