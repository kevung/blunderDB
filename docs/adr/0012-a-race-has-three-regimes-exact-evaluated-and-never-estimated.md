# A race has three regimes: exact, evaluated, and never estimated

Status: accepted.
See also: ADR-0009, ADR-0027.

## Context
ADR-0009 forbids estimating a cube verdict from a summary statistic. An expectiminimax search
with a cube model (gammonNet, ADR-0011) does not summarise the trajectory, it plays it out, so
it can give a race verdict without the refuted method. The exact TS-06-06 table covers only
16.6 % of pure-bearoff positions in a real 88 015-position library: five races out of six had
no verdict at all.

## Decision
**A race answer carries one of three regimes, always named on screen.**

1. **exact** — read from a two-sided table. It wins wherever available and nothing displaces
   it; an engine's agreement with it is not the answer.
2. **evaluated** — produced by the engine's search. Available wherever the engine is,
   including at a match score.
3. **estimated** — the convolution and its correction (ADR-0009 rule 3), for the win
   probability only, with its error bound. Never a cube verdict (ADR-0009 rule 4).

**The exact table stays the floor.** It is the claim an evaluator cannot make, covers the most
studied domain (≤ 6 checkers a side), and is the oracle regime 2 is measured against. How it
reaches the machine: ADR-0027.

**Regime 2 ships with a number, not a claim.** Its agreement with the exact table is measured
and published in `doc/source/manuel.rst` next to the word "evaluated". Measured (gammonNet
2-ply `k=12`, 4 000 money cube decisions sampled over 1..6 checkers a side): verdict agreement
93.4 % overall, 61.1 % within 1 % of the take point rising to 94.4 % beyond 20 %; |Δ win
probability| mean 0.85 %, p95 3.21 %; |Δ cubeful equity| mean 0.039, p95 0.151, max 0.406. It
is money-only: the table has no gammon breakdown, so no exact oracle exists at a match score.

**One panel never shows two sources for the same quantity.** Where the exact figure exists it
is shown. The one seam is marked, not hidden: inside the exact domain the win probability is
read while candidate moves are ranked by the network, whose leaves do not consult the table
(ADR-0011).

## Consequences
- The regime badge has three values; the regime vocabulary lives in `CONTEXT.md`.
- Race cube verdicts exist at a match score.
- Disagreements concentrate at the take point, where two engines legitimately differ on a
  close call.
- Rejected: no verdict outside the exact domain — the panel would fall silent on races, where
  an engine is most reliable, while answering on any contact position.
- Rejected: the engine's verdict everywhere — trades a lookup for an evaluation.
- Rejected: dropping the two-sided table — it is the oracle; only a new ADR backed by a
  measurement could do it.

## Guard
`pkg/blunderdb/engine/gammonnet/eval_measure_test.go` (`BLUNDERDB_EVAL_MEASURE=1`; extends to
TS-06-11 when `BLUNDERDB_TS11_PATH` is set).
