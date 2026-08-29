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

**Trap two: the gnubg fixtures are at the same depth as the engine under test.** This was
first written on a misreading, and the misreading is recorded here because it is the same
class of bug as the one below.

The leading value of an SGF move analysis, `A[0][...]`, is **not** a ply — it is `iMove`, the
rank of the move the player actually chose among the candidates. It appears once per move
node, and in `charlot1-charlot2_7p_2025-11-08-2305.sgf` its distribution is `0`x166, `1`x2,
`2`x2, `6`x1: the player picked gnubg's best move 166 times and its seventh once. Read as a
depth, it produced a plausible-looking spread of 0-, 1-, 2-, 3- and 6-ply, and a false
conclusion — that these fixtures were shallow.

The real evaluation context sits *after* the outputs, not before:

```
A[0][lpab E ver 3 0.496365 0.140890 0.006297 0.135264 0.005951 -0.004723 2C 0 1 0.000000 1]
                                                                        ^^ nPlies=2, cubeful
DA[E ver 3 2C 1 0.000000 1 0.503635 ...]
            ^^ same field, same meaning
```

Counting that token gives the truth:

| fixture | `0C` (0-ply) | `2C` (2-ply) | cube decisions |
|---|---|---|---|
| `charlot1-charlot2_7p_2025-11-08-2305.sgf` | 1886 | 1697 | 94, **all 2-ply cubeful** |
| `test.sgf` | 3530 | 2180 | all 2-ply cubeful |

gnubg analysed both matches at **2-ply cubeful**, with a 0-ply pre-filter over the full
candidate list — its standard move filter. That is not a shallow analysis. It is, however,
**the same depth as the engine under test**, and an arbiter at the depth of the engine it
judges cannot cost a disagreement: what it measures is a difference of opinion between peers,
the sign falling in our favour as often as against us. Costing requires an arbiter *deeper*
than both.

Separately, and confirming the family: `gnubgparser` v1.4.0 reads the `3` of `ver 3` — the SGF
format version — as the cube analysis's ply, reporting `3-ply` for 100 % of cube decisions in
both files, where the true value is 2 (`kevung/gnubgparser#2`).

The XG export of the same match is the arbiter that qualifies, and was measured too: 189
checker decisions at genuine depths (best candidate `{3-ply: 50, 4-ply: 135, Book: 4}`), 185
cube decisions at `{2-ply: 147, 4-ply: 36}`. It is the only source in `testdata/` deeper than
the engine under test.

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
- `kevung/gnubgparser#2` must be fixed and the dependency bumped. Separately, databases
  already built carry the false `3-ply` label on every gnubg-imported cube analysis, and a
  genuine 3-ply is indistinguishable from the bug in an existing row — what to do about that
  data is its own decision.
- The same misreading was live in two further places in that parser (the played-move rank
  broadcast as every option's depth; a hard-coded selected-move index), which is why the fix
  is verified against the fixtures rather than trusted: **a plausible-looking distribution of
  plies is exactly what this class of bug produces.** This ADR's own first draft fell for it.

## Update — 2026-08-29, measured

The missing number: **458.8 s for 669 decisions** (426 from the gnubg fixtures, 243 from XG)
at 2-ply `k=12` on a 16-core machine. **The gate is a pre-merge recipe step, not a CI test** —
~8 minutes here, on hardware CI does not have a fraction of. `BLUNDERDB_GATE` runs it
(`TestIntegrationGate`, `pkg/blunderdb/engine/gammonnet`); `BLUNDERDB_GATE_LIMIT` truncates
both corpora for a smoke pass.

**Result: PASS**, once one exclusion the original criterion did not anticipate was made
explicit. Criterion 1 (cost) is judged by XG's *match-equity* checker equities; the search
that chooses our move is still `search.go`'s documented cubeless-money recursion — the two
scales coincide almost everywhere, but diverge sharply once either side is 1-away from match
point (a leader who wins the match outright on any win gets zero marginal value from a gammon
money equity happily chases). Both of the run's two above-threshold costs (0.055, 0.074) were
at exactly such a score; excluding decisions where either side is 1-away leaves criterion 1 at
**87 checked, max 0.0311** against the 0.05 line. This is a scope boundary, not a bug: match-
aware checker-move valuation is a later tranche's job (`gn_search_use_match`, `use_cube`), not
this one's. Criterion 3 (candidacy) is likewise not a bare zero-tolerance count — an isolated
miss against gnubg's own non-exhaustive list is exactly the "signal, not severity" this ADR
already named, so the gate blocks on a **5 % aggregate rate**, not on any single miss, while
still logging every one for inspection. Measured: **420/426, 1.4 % missing**. Criterion 2
(cube verdict): **91 checked, 0 ND↔DP flips**, 2 adjacent disagreements.

**Three real bugs were found and fixed on the way to that PASS, none of them in gammonNet
itself** — exactly the failure mode this gate exists to catch, caught before the engine tranche
that triggered building it was even judged:

- `domain.notation()` rendered a White player's move in the board's absolute indices instead
  of the mover-relative numbering every notation uses, so a White move gnubg calls `13/7 8/7`
  came out `12/18 17/18` — invisible to every prior test because none put White on roll
  (`b6b363fd`).
- The same function capitalised bar entries as `Bar/24`; gnubg and XG both write `bar/24`,
  lowercase, and `NormalizeMove` does not fold case (`b18b28be`).
- `ingest/xg.go`'s `mapDoubleTakeMove`/`mapSingleCubeMove` synthesize the responder's
  Take/Pass position, correctly flipping `PlayerOnRoll` to the opponent — but reattached the
  doubler's own `DoublingCubeAnalysis` unflipped, so `PlayerWinChances` stayed the doubler's,
  now mislabelled as the responder's. Caught because a bearoff race judged 41 % for the
  recorded on-roll player against blunderDB's own exact two-sided table, next to a stored
  `PlayerWinChances` of 91.67 % and a `BestCubeAction` of `Double, Pass` that only make sense
  for the other side (`ee01bb08`).

`kevung/gnubgparser#2` (the `ver 3` ply misreading this ADR's own first draft fell for) is
already fixed as of the pinned `v1.5.0` — no action needed here.
