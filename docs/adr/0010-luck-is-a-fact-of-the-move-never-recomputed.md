# Luck is a fact of the move, read from the source file, never recomputed

Status: accepted.

## Context
Luck is per roll: the equity of the best play with the dice rolled minus its expectation over
all 36 rolls (gnubg `LuckNormal`, 0-ply cubeful). eXtreme Gammon stores it as `ErrLuck` on each
move record, gnubg SGF as the `LU` property; BGF and Jellyfish `.mat` carry none. Positions are
deduplicated by Zobrist hash and their analyses merged, so where the value lives matters.

## Decision
1. One nullable column, `move.luck_mp` (signed millipoints of EMG, positive = lucky), filled at
   import from the source file's own value and never recomputed.
2. **On `move`, not `analysis` or `position`.** Luck is a fact of one occurrence of a roll in one
   match; a Position is a shared identity. Storing it on the merged analysis would leak one
   match's luck into another's statistics. Forced rolls have a `move` row but not always an
   analysis, and luck aggregates count every roll.
3. **NULL means unknown, never zero.** Zero is a neutral roll. BGF/`.mat` imports, unanalysed
   rolls and rows imported before the column existed are NULL and excluded from both numerator
   and denominator of any luck rate.

## Consequences
- Per-player luck rates divide by the count of luck-carrying rolls, not total rolls.
- An existing database shows no luck until its matches are re-imported: luck is not in the
  stored analysis, so no repair can rebuild it.
