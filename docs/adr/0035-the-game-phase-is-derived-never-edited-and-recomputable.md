# The game phase is derived, never edited, and recomputable

Status: accepted.

## Context

A player who wants to know whether they lose points in the race or in contact needs a column
that says which, readable by the search and by the statistics (which do not run the search
grammar). The only sourced, deterministic classification boundaries are gnubg's
(`ClassifyPosition`: over, race, crashed); every "plan" a player names — holding, backgame,
blitz, prime-vs-prime — sits in gnubg's single contact class with no published threshold.

## Decision

1. **`position.game_phase` is a derived label**: opening, middlegame, race, bearoff, plus
   `unknown` for a row not classified (or whose state cannot be decoded — written, not skipped).
2. **It is computed from the board alone**, by `engine.ClassifyGamePhase`: not from the move
   number (one Zobrist row is reached at different moves in different matches), nor from the
   cube, score or side on roll. The classification is symmetric.
3. **Contact vs race is `domain.Position.MatchesNoContact`** (gnubg's crossing test); bearoff
   is every checker still on the board standing in its own home board.
4. **The opening boundary is a named convention**: `engine.OpeningDisplacementMax` = 4, the
   most checkers either side may have moved off its starting points (two ordinary rolls, or
   one doublet). Nothing else compares against 4.
5. **It is never editable.** No command, panel or route sets a phase.
6. **It is recomputable.** `blunderdb repair` reclassifies every row that disagrees with the
   classifier; the schema migration that introduced the column runs the same pass.

## Consequences

- The `ph` search token and the statistics' per-phase figures read one indexed column.
- The label is a convention, not a standard, and the user-facing documentation says so.
- The user cannot correct a position; in exchange a rule change is applied to every row at once.
- A plan classifier (holding, backgame…) is a separate, larger decision (#291), not this label.
- Rejected: composing a phase from search filters (`nc` + pip range + off count) — wrong at the
  edges and unavailable to the statistics.
- Rejected: an editable column — two sources of truth, and later rule changes become unappliable.

## Guard

`pkg/blunderdb/engine/gamephase_test.go`.
