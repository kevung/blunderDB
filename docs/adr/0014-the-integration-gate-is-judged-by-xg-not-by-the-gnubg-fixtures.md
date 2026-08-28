# The integration gate is judged by XG, not by the gnubg fixtures

## Status

accepted — decided 2026-08-28

## Context

A ported evaluator must not be merged on the strength of its unit tests. Network parity and
search parity (ADR-0011) prove the port reproduces the C reference; they do not prove the
whole chain — codec, search, MET, cube — says sensible things about a real match at a real
score with a real cube.

The obvious gate is to replay an analysed match from `testdata/` and compare. Two traps sit
in the way, and both were measured before this was written.

**Trap one: agreement rate is the wrong criterion.** gammonGo's ADR 0097 refuses "the XG
judge" for exactly this reason — conformity on close positions measures an engine's
*strength*, already measured upstream, and produces a permanent false red where two engines
legitimately differ. That objection is sound against a criterion built on the *count* of
disagreements. It does not hold against one built on their *cost*: a botched port does not
disagree more often, it disagrees **expensively**. A colour inversion, a flipped perspective,
permuted dice or a silently ignored score all cost 0.1–0.5 equity, never 0.005. Upstream
published exactly that shape — over 139 decisions of a 7-point match, 86.3 % agreement with
gnubg, the 19 disagreements costing a median of +0.0048, a maximum of +0.0195, none above
0.05.

**Trap two: the gnubg fixtures are not deep enough to be an arbiter.** Measured on the
fixtures themselves, not assumed:

| fixture | checker decisions | depth at the best candidate |
|---|---|---|
| `charlot1-charlot2_7p_2025-11-08-2305.sgf` | 180 | **0-ply: 166**, 1-ply: 2, 2-ply: 2, 3-ply: 9, 6-ply: 1 |
| `test.sgf` | 311 | **0-ply: 204**, 1-ply: 57, 2-ply: 17, rest scattered |

A disagreement cannot be costed by an arbiter shallower than the engine under test; the sign
would fall in our favour as often as against us. Worse, the depth those files report for
*cube* decisions is not a depth at all: the SGF property `DA[E ver 3 ...]` carries no leading
ply bracket the way `A[0][... ]` does, and `gnubgparser` v1.4.0 reads the `3` of `ver 3` — the
format version — as the ply. It is `3` for 100 % of cube decisions in both files
(`kevung/gnubgparser#2`).

The XG export of the same match is a different matter, and was measured too: 189 checker
decisions at genuine depths (best candidate `{3-ply: 50, 4-ply: 135, Book: 4}`), 185 cube
decisions at `{2-ply: 147, 4-ply: 36}`. It is the only arbiter in `testdata/` deeper than the
engine under test.

## Decision

**XG is the arbiter of cost and of cube verdicts. The gnubg fixtures serve a different,
narrower purpose.** Using XG here does not contradict ADR 0097: that ADR refused XG as a
*measure of strength*; this gate uses it as a *cost oracle for integration bugs*, which is
the job it is fit for.

Three criteria block a merge:

1. **No checker disagreement costs more than 0.05 equity**, judged by XG's own stored
   equities. Two and a half times the worst case published upstream: it cannot close on
   engines merely differing.
2. **No cube verdict disagreement of ND↔DP**, in either direction. Adjacent disagreements
   (ND↔DT, DT↔DP) are boundary noise — two engines placing the take point half a percent
   apart. ND↔DP never is. The criterion draws its power from an unexpected side: XG's
   distribution is `{ND: 171, DT: 12, DP: 2}`, so there are only two chances to miss a pass —
   but **171 chances to double wrongly**, and an inverted perspective or an ignored score
   doubles wrongly in bulk. This is the only test that catches a score silently dropped
   across codec → search → MET.
3. **Every chosen move appears among the arbiter's candidates.** This one runs against the
   *gnubg* fixtures, which store on average **19.4** candidates per decision (min 1, max 221),
   each with its equity and full six-way probability breakdown. A move absent from a net that
   wide is a signal, not a severity. XG stores 7.2 on average (max 10) — too tight for this
   criterion, which is why the two corpora are not interchangeable.

**Reported, never blocking**: the best-move agreement rate (reference 86.3 %) and the PR
(reference 0.375, CI [0.264; 0.499] at 2-ply `k=12`). One match is a small sample for a PR;
these inform, they do not decide.

**Where the gate runs is decided by measurement, not now.** 781 analysed decisions across the
two SGF fixtures, 1155 including the XG file. The per-decision cost of a 2-ply search in Go
does not exist yet; it is the first number the port must produce, and it alone decides between
a CI test and a pre-merge recipe step.

## Considered options

- **A blocking floor on the agreement rate** (e.g. ≥ 80 %). Rejected as a blocking criterion:
  the rate depends on the match — a race-heavy or bearoff-heavy corpus shifts it with nothing
  broken — so the threshold would need rejustifying whenever the corpus changed.
- **A PR gate against the arbiter.** One directly comparable figure, and the control upstream
  calls most revealing. Rejected as blocking: 189 decisions against upstream's 600 gives an
  interval too wide to discriminate.
- **gnubg as the arbiter**, the obvious first choice. Rejected on the measurement above.

## Consequences

- The gate needs both corpora, for different reasons; neither replaces the other.
- It proves the port did not damage the network. It does not prove the network is good —
  that was measured upstream, and its verdict is quoted as rendered: *equivalent to GNU
  Backgammon at 2-ply, confirmed*. "Superior" is not established, and eXtreme Gammon was
  never measured.
- `kevung/gnubgparser#2` must be fixed and the dependency bumped; separately, databases
  already built carry the false `3-ply` label on every gnubg-imported cube analysis, and what
  to do about existing data is its own decision.
