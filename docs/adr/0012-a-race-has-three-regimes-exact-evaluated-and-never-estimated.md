# A race has three regimes: exact, evaluated, and never estimated

## Status

accepted — decided 2026-08-28. Extends ADR-0009, which it does not contradict.

## Context

ADR-0009 established the Bearoff panel's dual regime. Inside a loaded two-sided database's
domain the panel is *exact* — a lookup. Outside it, the win probability is *estimated*
(convolution of the two one-sided roll distributions plus a frozen calibrated correction:
σ = 0.05 %, p99 = 0.15 %, max 0.42 %) and shown with its error bound, **and the cube verdict
is absent**. That absence was not caution; it was measured:

- The best static model tried — p plus both distributions' mean, σ, skew and kurtosis, 45
  degrees of freedom — left RMS 0.016 equity, max 0.20. *"Cube equity is a timing problem
  along the trajectory; no snapshot summary carries it."*
- Converting to a match score through the MET is worse: over 5280 gnubg evaluations, the
  dead-cube `p → MET` chain agreed on D/ND only **87.8 %** of the time, with real blunders;
  even gnubg's own full 0-ply cube model reached only **96.5 %**.

ADR-0009 concluded that "the dual regime **is** the interface", and that there is no future
state in which the panel is always exact.

A ported evaluator changes one term of that argument, and only one. What was refuted was
*estimating a cube verdict from a summary statistic*. An expectiminimax search with a cube
model does not summarise the trajectory — it plays it out. The gap ADR-0009 left open can now
be filled without rehabilitating the method it refuted.

The scale of the gap is worth stating: the embedded TS-06-06 covers **16.6 %** of pure-bearoff
positions in a real 88 015-position user database. Five races out of six already fell outside
the exact regime and got no verdict at all.

## Decision

**A race answer carries one of three regimes, always named on screen.**

1. **exact** — read from a two-sided database. It wins wherever it is available, and nothing
   displaces it. An exact figure is *the* answer; an engine's agreement with it is not.
2. **evaluated** — produced by the engine, which plays the trajectory. Available wherever the
   engine is, including **at a match score**, which ADR-0009 declined to offer because the
   chain it had was the `p → MET` one.
3. **estimated** — the convolution plus its calibrated correction. It keeps its place for the
   *win probability*, with its error bound, and it remains **forbidden for a cube verdict**.
   ADR-0009's prohibition stands, unamended, against the method it named.

**The two-sided database stays embedded.** Its unique contribution narrowed once regime 2
exists, but "exact" is a claim an evaluator cannot make, the domain it covers (≤ 6 checkers a
side) is the most studied in the literature and the one where a user can check blunderDB
against a book — and it is the **oracle** by which regime 2 is measured at all. Demoting it
to a download would save 6.8 MB and remove no complexity: the dual regime stays in the code
either way, since the table returns the moment it is downloaded.

**Regime 2 does not ship with a claim, it ships with a number.** Before the evaluated regime
is offered, a measurement compares the engine against the exact table where both answer, and
reports: the money cube verdict agreement rate, broken down by distance to the take point;
the distribution of |Δ win probability|; the distribution of |Δ cubeful equity|. That number
is published in the documentation next to the word "evaluated". Inventing a regime and asking
users to trust it is not an option.

**One panel never shows two sources for the same quantity.** Where the exact figure exists it
is the figure shown. A known seam remains and is marked rather than hidden: on a race inside
the exact domain, the win probability is *read* while the candidate moves are *ranked by the
network*, since the search's leaves deliberately do not consult the table (ADR-0011).

## Considered options

- **Leave ADR-0009 untouched**: no verdict outside the exact domain, even when the engine has
  one. Rejected: the panel would give a cube verdict on any contact position and fall silent
  precisely on races, the domain where an engine is most reliable. A user reads that as a
  failure, not as rigour.
- **Let the engine's verdict win everywhere**, one regime, no explanation needed. Rejected:
  it trades a lookup for an evaluation and discards what the ts-bearoff work built.
- **Drop the two-sided table** now that an engine answers everywhere. Rejected on the grounds
  above; if the measurement ever justifies it, it is a separate amending ADR, not a side
  effect of this one.

## Consequences

- The regime badge grows a third value; the download hint stays attached to regime 1 only.
- Race cube verdicts become available at a match score for the first time.
- The measurement is a prerequisite of shipping regime 2, and its numbers belong in
  `doc/source/manuel.rst` beside the methodology section.
- The regime vocabulary enters `CONTEXT.md`.
