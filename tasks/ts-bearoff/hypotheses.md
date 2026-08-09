# Hypothesis catalogue — every number the panel shows, and what it assumes

This file is the source of truth for the user-documentation methodology
section (PR 3, French). Nothing here may be dropped in transcription: the
project committed (ADR-0009) to stating every hypothesis behind every
displayed value. Keep the structure: *what is shown → how it is computed →
what it assumes → how wrong it can be*.

## Domain of the panel

Both players have **all remaining checkers in their own home board** (pure
bearoff). Checkers on the bar or outside the home board put the position out
of domain: the race zone shows nothing (the EPC blocks already handle the
partial case today). Extending to open no-contact races is explicitly out of
scope (no oracle → no honest error bound; ADR-0009).

## EPC block (per player) — always exact

- **Shown**: EPC, mean rolls, σ of rolls, pip count, wastage.
- **Computed**: distribution of the number of rolls to bear off, read from
  the embedded one-sided database `gnubg_os6.bd` (OS-06-15, covers all
  15 checkers). EPC = meanRolls × 49/6; wastage = EPC − pip.
- **Assumes**: gnubg's one-sided optimal play — the player minimises their
  own bear-off rolls, ignoring the opponent entirely. 49/6 ≈ 8.1667 is the
  exact average pips per roll (doubles counted four times).
- **Error**: none within the assumption; the distribution is exact. The
  one-sided-play assumption itself is the only idealisation, and it is the
  standard definition of EPC.

## Win probability p (race zone) — two regimes

### Exact regime — badge « exact »

- **Shown**: p for the player on roll.
- **Computed**: direct lookup, plane 0 (cubeless) of the widest available
  two-sided database (embedded TS-06-06 / user file / downloaded TS-06-11).
  Scale `u/65535` for p (equity `u/32767.5 − 1`).
- **Assumes**: both sides play the true two-sided optimal checker play (the
  database is computed by full retrograde analysis). No further hypothesis.
- **Error**: quantisation only (< 1/65535 ≈ 0.0015 %).

### Estimated regime — badge « estimé », bound displayed

- **Shown**: p ± bound.
- **Computed**: convolution of the two players' one-sided roll distributions
  (player on roll wins iff their roll count ≤ opponent's), then a frozen
  polynomial correction (32 coefficients, function of p and both
  distributions' moments) calibrated offline against the TS-06-11 oracle.
- **Assumes**:
  1. **Independence** of the two bear-off processes — true in bearoff
     (no contact, no blocking), this is structural, not an approximation;
  2. **One-sided optimal play by both sides** — this is the approximation:
     in reality the trailer deviates to play for variance and the leader
     for safety. The measured effect is an antisymmetric bias (convolution
     overstates the leader's edge) of up to ±0.11 % of win probability;
  3. The **correction** absorbs that bias statistically. It was calibrated
     and validated on the 9–11-checker slice of the TS-06-11 oracle.
- **Error (measured, after correction)**: σ = 0.048 %, p99 = 0.15 %,
  max observed = 0.42 % of win probability. Error *decreases* as checkers
  increase past 6. **Beyond 11 checkers per side the bound is an
  extrapolation** — monotone trends support it, no oracle certifies it
  (follow-up: TS-06-12 via `makebearoff`). The documentation must say
  "extrapolé au-delà de 11 dames" verbatim.

## Money equities and cube verdict (race zone) — exact regime only

- **Shown**: ND / DT / DP cubeful equities and the verdict (No double /
  Double-Take / Double-Pass), **money game**.
- **Computed**: planes 1–3 of the two-sided database + the standard money
  cube logic for the TS class (as gnubg implements it).
- **Assumes**: money game, **no Jacoby**, gammonless — within ≤ 11 checkers
  per side gammons are impossible (both sides have borne off ≥ 4), so this
  is not an approximation there; it becomes one only if a wider database is
  ever used on 12–15-checker positions.
- **Error**: quantisation only.
- **Why money and not the match score**: converting verdicts through a MET
  was measured against gnubg 2-ply on 5 280 score/cube scenarios: the simple
  dead-cube chain mis-decides D/ND 12 % of the time (worst case ≈ 17 % MWC —
  it ignores recubes and the option value of waiting); even gnubg's full
  0-ply cube model disagrees with 2-ply in 3.5 % of cases. A verdict that
  wrong is worse than no verdict. The money verdict is the referential of
  the bearoff literature and is exact.

## Absence of a verdict (estimated regime) — deliberate

- Outside the exact domain the verdict line is **absent**, not greyed:
  cubeful equity does not estimate from any snapshot summary (best static
  model measured: RMS 0.016 equity, max error 0.20 — enough to flip every
  marginal decision). The panel offers the TS-06-11 download instead
  (« videau exact jusqu'à 11 dames »).

## Sources and their provenance (for the doc's practical section)

- Embedded: `gnubg_ts0.bd` (TS-06-06, 6.8 MB), gnubg's own default database.
- User file: any gnubg two-sided `.bd` (config path); widest domain wins.
- Download: TS-06-11 (1.23 GB), GitHub release tag `bearoff-data-1`,
  SHA-256 verified, stored in `$XDG_DATA_HOME/blunderdb/`, deletable from
  the same dialog. Regenerable from scratch with gnubg's `makebearoff`.

## Défi mode (doc wording)

Défi masks results per zone (bottom EPC / top EPC / race) behind
« cliquer pour révéler » overlays; any edit to the position re-masks all
zones. Badges are masked with their zone (knowing the answer is "exact"
already leaks the checker count range). State persists in config.
