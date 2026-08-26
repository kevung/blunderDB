# Luck is a fact of the move, read from the source file, never recomputed

## Status

accepted — design decided 2026-08-26 (issue #116 grilling session),
implementation tracked in `tasks/issue-116-onglet-joueurs.md`

## Context

Issue #116 asks for a per-player summary table including luck. Luck is a
per-roll quantity: the equity of the best play with the dice actually rolled,
minus the expectation of that equity over all 36 rolls (gnubg computes it with
a 0-ply cubeful evaluation at analysis time — `gnubg/analysis.c`,
`LuckNormal`). blunderDB has no full evaluation engine, so it cannot compute
luck itself; but the analysed source files carry it — eXtreme Gammon stores
`ErrLuck` on each move record, gnubg SGF stores the `LU` property per roll.
BGF and Jellyfish `.mat` files carry none.

blunderDB stores nothing luck-related today. The question was where the value
belongs, given that Positions are deduplicated by Zobrist hash across imports
and their analyses merged.

## Decision

One nullable column, `move.luck_mp` (signed millipoints of EMG, positive =
lucky), filled at import from the source file's own value and never
recomputed:

- **On `move`, not `analysis` or `position`.** A Position is an identity,
  deduplicated and shared across matches; the luck of a roll is a fact of the
  *occurrence* of that roll in one match — the same occurrence/identity split
  the domain already draws between `move` and `position`. Storing it on the
  merged analysis would let one match's luck leak into another's statistics.
  Practically, forced rolls have a `move` row but not always an analysis, and
  luck aggregates count every roll.
- **NULL means unknown, never zero.** Zero is a real value (a neutral roll).
  Rows imported before this column existed, BGF/`.mat` imports, and
  unanalysed rolls are NULL and are excluded from both the numerator and the
  denominator of any luck rate.

## Consequences

- Schema bump (DatabaseVersion 2.15.0), migrated on both backends.
- Existing databases show no luck until their matches are re-imported from the
  source files. The #115-style repair of denormalised columns cannot help:
  luck is not in the stored analysis JSON, so there is nothing on disk to
  rebuild it from.
- Per-player luck rates divide by the count of luck-carrying rolls, not total
  rolls — an honest average over what is actually known.
