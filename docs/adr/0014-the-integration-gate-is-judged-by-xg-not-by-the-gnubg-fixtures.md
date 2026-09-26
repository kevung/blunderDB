# The integration gate is judged by XG, not by the gnubg fixtures

Status: accepted.
See also: ADR-0011

## Context

Network and search parity (ADR-0011 rule 6) prove the port reproduces the C; they do not prove
the whole chain — codec, search, MET, cube — says sensible things about a real match. A botched
integration (colour inversion, flipped perspective, permuted dice, ignored score) does not
disagree *more often*, it disagrees *expensively*: 0.1–0.5 equity, where upstream's genuine
disagreements with gnubg cost at most 0.0195. The gnubg fixtures in `testdata/` were analysed at
2-ply cubeful — the engine's own depth — so they cannot cost a disagreement; the XG export of the
same match (3-/4-ply checker, 2-/4-ply cube) is the only deeper arbiter.

## Decision

XG is the arbiter of cost and of cube verdicts; the gnubg fixtures serve candidacy only. Three
criteria block a merge:

1. **No checker disagreement costs more than 0.05 equity**, judged by XG's stored equities, at
   every score (no 1-away exclusion).
2. **No ND↔DP cube-verdict disagreement**, either direction. Adjacent disagreements (ND↔DT,
   DT↔DP) are boundary noise. This is the criterion that catches a score silently dropped.
3. **Chosen moves appear among the gnubg fixtures' candidates** (19.4 per decision on average;
   XG's 7.2 is too tight), blocking on an aggregate **5 % missing rate**, every miss logged.

**Reported, never blocking**: best-move agreement rate and PR against the arbiter.

The gate is a **pre-merge recipe step, not a CI test**: 669 decisions at 2-ply take ~8 minutes
on 16 cores.

## Consequences

- The gate needs both corpora, for different reasons; neither replaces the other.
- It proves the port did not damage the network, not that the network is good — upstream's
  verdict stands as published: *equivalent to GNU Backgammon at 2-ply*.
- Two checker decisions at a 1-away/5-away Crawford score fail criterion 1 (0.0552, 0.0738).
  Neither `use_cube`, cube efficiency, depth, pruning nor MET moves them (ADR-0029); they are
  the network's judgement on two boards. The gate is **left failing, never loosened** to hide
  them.
- Rejected: **a blocking agreement-rate floor** (depends on the corpus's mix, not on
  correctness); **a blocking PR** (189 decisions give too wide an interval); **gnubg as
  arbiter** (same depth as the engine under test).

## Guard

`TestIntegrationGate` in `pkg/blunderdb/engine/gammonnet/integration_gate_test.go`, run with
`BLUNDERDB_GATE=1` (`BLUNDERDB_GATE_LIMIT` for a smoke pass); its header records the latest run.
