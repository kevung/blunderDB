# Lot 2 — the EPC beyond the home board

ADR-0027 §9, second lot. Depends on lot 1 (the one-sided generator already accepts *n* points).
Branch `feat/epc-beyond-home`.

## What changes for the user

Today the EPC block of the panel is silent unless every checker of a side is in its home board.
After this lot, a side whose farthest checker stands on point ≤ *p* gets its EPC from an
*n*-point one-sided table (`gnubg_os<p>.bd`) generated from the Bearoff tab, exactly like a
larger two-sided domain. The convolution estimate of the win probability (ADR-0009) widens with
it, since it is built from the two one-sided distributions.

## Steps

- [ ] `engine/epc.go`: `BearoffDatabase` reads `nPoints` from the header instead of assuming 6;
      `positionBearoff` and `ComputeEPC` take a `[]int` of length `nPoints` (the `[6]int`
      signatures stay as thin wrappers for the existing callers and tests).
- [ ] `engine/race/epc.go` `computeSide`: `AllInHome` becomes `WithinPoints(p)`; the home-board
      array becomes the table's width. Keep `Side.AllInHome` in the JSON for the panel's existing
      logic (a position with checkers outside still has no two-sided verdict).
- [ ] `engine/race/convolve.go`: the estimate takes distributions of any width; recalibration
      question — the correction coefficients (`correction_coeffs.go`) were fitted on the 6-point
      domain. Measure the raw convolution error on a 7–10-point oracle before deciding whether the
      correction applies, is refitted, or is disabled with a wider bound beyond six points.
      **Do not extrapolate silently**: the bound shown must be the one measured.
- [ ] Panel: EPC block shows the table's domain next to the regime ("exact, OS-08"); no verdict
      change.
- [ ] Bearoff tab: the picker gains the one-sided axis (points 6 … 12; size, RAM, time as for
      two-sided). Fingerprints for OS-07 … OS-10 from lot 1 §2.
- [ ] CLI: `bearoff generate --os 8`.
- [ ] Tests: identity `TestOneSided_7_IdenticalToGnubg` (oracle dir); EPC of a position with a
      checker on the 8-point against gnubg's `show epc`-equivalent (fixture from gnubg CLI).
- [ ] Docs: `manuel.rst` EPC section + eight `.po`.

## Open question to settle before starting

Whether a race position with checkers outside both home boards but inside an *n*-point
one-sided table should also get an **evaluated** cube verdict from gammonNet (ADR-0012 already
allows it outside the two-sided domain) with the *exact* EPC next to it — or whether the panel
keeps today's rule that the race zone appears only for pure bearoffs. Recommendation: keep the
rule for this lot; widening the race zone is a display decision under ADR-0017/0021, not a data
one.
