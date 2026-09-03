# The cube's level inversion becomes a closed form, and that is written upstream

## Status

accepted — decided 2026-09-03. Closes #194 (plan 2026-09b, fiche C.7). Lands on *"The live
cube curve runs to the average win"* (0022, which gave the level curves the shape this record
inverts) and on *"The search values its leaves with the cube"* (0023, which made every leaf of
the search pay for that inversion). **Changes no behaviour here**: it records a decision,
measures it, and says where the change has to be written.

**Grouped with the upstream half of *"Cube efficiency is measured per cube state, and read at
the root"* (0029, point 4).** Both are changes to `gn_cube.c`, both are Configuration changes in
`CONTEXT.md`'s sense, and both invalidate exactly the same things: a new gammonNet tag, a new
`EngineVersion`, `testdata/cube_gold.bin` and `testdata/search_cube_gold.bin` regenerated from
the fixed C, and every stored gammonNet analysis stale. Landing them separately would spend that
bill twice. This record therefore does not propose its own release; it proposes its own patch,
to be cut with the other one.

## Context

`levelSolve` (`pkg/blunderdb/engine/gammonnet/cube.go`, upstream `level_solve` in `gn_cube.c`)
inverts the match cube model's stake-level curve: given a target MWC, it returns the winning
probability at which the curve reaches it. It does so by **sixty steps of bisection**.

The function being inverted is **piecewise linear and monotone**, and its two or three pieces are
known before the first step — they are exactly the segments `levelLive` evaluates. Janowski's own
model gives the money breakpoints in closed form, and at a score the live take point is a finite
recursive product, so it is still a closed form;
`docs/recherche/P6-videau-janowski.md` establishes both (its Key Finding 5: *"la bissection est
superflue"*). Inverting a monotone piecewise-linear function is: pick the segment whose endpoints
bracket the target, then one division. **The bisection is not an approximation the model needs.
It is a numerical search where a division suffices.**

### What it costs — remeasured after C.8/C.9/C.10, not taken from the fiche

The fiche's profile predates the panel/parallelism work merged the same day, so every number
below was taken again. Ryzen 7 PRO 6850U, 16 cores, Go 1.25.13, 2026-09-03.

| benchmark | ns/op |
|---|---:|
| `BenchmarkLevelSolveBisection` (one inversion, 60 steps) | **126** |
| `BenchmarkLevelSolveClosed` (the same inversion, closed form) | **6.8** |
| `BenchmarkBuildLevelAnchors` (the chain's MET lookups alone) | 190 |
| `BenchmarkBuildLevels` (anchors **+** breakpoints) | 1 160 |
| `BenchmarkCubeDecisionMoney` (`Decide`, no chain: §3 closed form throughout) | 41 |
| `BenchmarkCubeDecisionAtScore` (`Decide` at 5-away/5-away) | 1 170 |

So the inversion is **×19** on its own, and the breakpoints are **84 %** of a stake chain
(1 160 − 190 of 1 160) — the anchors, six MET lookups per level, are the cheap half.

The profile of one canonical 2-ply k=12 decision at 5-away/5-away
(`BenchmarkDecision2PlyMatch`, `-cpuprofile`): `EvaluateBatch` 52.6 %, `Value` 39.8 %,
`buildLevels` 38.7 %, **`levelSolve` 35.4 % cumulative** (of which `laneCurve.at` 9.3 %). The
cube is the second cost centre of a decision at a score, and the bisection is almost all of the
cube.

End to end, with `levelSolve` patched to the closed form (patched, measured, reverted):

| | before | with the closed form |
|---|---:|---:|
| 2-ply decision, money (`BenchmarkDecision2Ply`) | 193 ms | 193 ms (untouched) |
| 2-ply decision, at a score (`BenchmarkDecision2PlyMatch`) | **306 ms** | **193 ms** |
| `Decide` at a score | 1 170 ns | ~520–720 ns |
| batch throughput, 0-ply, 32 real positions × 16 workers | 4 500 pos/s | **6 780 pos/s** |
| batch throughput, 2-ply | 12.2 pos/s | **15.6 pos/s** |

A decision at a score comes to cost what the same decision costs in money: the whole of the
match cube recursion's overhead disappears. ×1.58 on the decision, ×1.28 on the analysis batch —
which is where it is counted in hours, on a database of 88 000 positions.

### What it changes — and this is the decisive part

`TestClosedFormAgreesWithBisection` (committed, always on) replays 725 328 inversions on real
stake chains — every away score from 1-away/1-away to 15-away/15-away plus the horizon case,
four cube values, Crawford and not, six outcome mixes, three cube states, six blends, and both
the real targets (`pass`, `cash`) and a sweep. The two agree to **3.4e-13** in p, worst case.
They are not the same bits:

- 42.7 % of inversions are bit-identical; worst |Δp| 2.25e-13; worst ULP distance 4 087 away
  from the bounds.
- Propagated through `Value`: 93.7 % of valuations bit-identical, worst |Δ| **4.4e-16** of
  normalised equity — two ULP.
- **The cube gold file reports `max|Δ| = 0.000e+00` today over 2 320 decisions.** The port is
  currently bit-for-bit identical to `gn_cube.c`. With the closed form in Go alone it becomes
  **1.665e-14** (on `take_point`).
- The search gold is unmoved to four significant digits in both configurations: 7.153e-07
  (money-cubeless, 85 decisions) and 3.902e-07 (match-and-cube, 126 decisions) against the C.
- The integration gate's output is **byte-identical** across 669 real decisions: same moves,
  same two red costs (0.0552 and 0.0738), same 5 candidacy misses, same 2 adjacent cube
  disagreements.

So: nothing a user sees moves, and the last two bits of a stored equity do.

## Decision

1. **The inversion becomes a closed form.** The gain is 35 % of a decision at a score and ×19 on
   the function itself, and it comes from replacing sixty steps of a serial dependency chain with
   one division. That is the shape of the algorithm, not its transcription: it survives a change
   of language, which is the criterion `CLAUDE.md` and gammonNet's ADR-0003 state. It is
   therefore **conceptual**, and conceptual optimisations are written in gammonNet first, with
   their measurement, and this port follows.

   The contrast with its two neighbours in the same file is the whole point of the criterion.
   `laneCurve` (hoisting the segment constants out of the sixty steps, ×1.24–1.51) is an
   *implementation* win: it exists because the Go compiler refuses to inline `levelLive` where
   gcc inlines `level_live`, it has nothing to send upstream, and it stays here. The ported
   `gn_cube_value_batch` (`cube_batch_experiment_test.go`) is the mirror image: a conceptual win
   *upstream* that this port measured at ×0.89 and refused. Neither is what this is.

2. **Nothing in this repository changes today.** A local rewrite would take the cube gold from an
   exact 0 to 1.665e-14 — which is to say the gold file would start measuring a divergence
   between this port and `gn_cube.c`, which is the one thing it exists to catch, and
   `cube.go`'s own header forbids ("a discrepancy from it is a bug, never an improvisation").
   Regenerating the gold to accommodate a local change destroys what it measures; the
   cube-efficiency record already rejected exactly that move, for exactly that reason.

3. **It is a Configuration change, not a free win.** The result is not bit-identical, so under
   *"The evaluator batches positions … and keeps the scalar sum order"* (0024) — where
   reproducibility of a stored analysis on another machine is what makes "stale" mean anything —
   the change gets a new `EngineVersion` naming the gammonNet tag that carries it, and every
   stored gammonNet analysis is stale and re-run by `AnalyzeStaleGammonNet`. The fiche's
   hopeful "`EngineVersion` unchanged if the result is bit-identical (it should be)" is
   answered: it is not, and 4.4e-16 on a leaf is still a different number in a database.

4. **It ships with the branch-local efficiency, in one tag.** Both changes are in `gn_cube.c`,
   both are Configuration changes, both regenerate `cube_gold.bin` and `search_cube_gold.bin`,
   both make every stored analysis stale. One upstream release, one port, one stale sweep.

5. **What the upstream change is** (delivered as a proposal, not applied):

   - `level_solve(level, owner, blend, target)` keeps its signature and loses its loop. It asks
     the level for its segments — `(x0, y0, x1, y1)` triples in ascending p, exactly the ones
     `level_live` selects between, dead level included — then walks them: a degenerate segment
     (`x1 <= x0`) is skipped, a target at or below the first live endpoint returns `x0`, a
     target inside `[v0, v1]` returns `x0 + (x1 − x0)·((target − v0)/(v1 − v0))`, and falling off
     the end returns 1. With `blend >= 0` the endpoints are blended first
     (`(1−blend)·M_dead(x) + blend·y`), which is exact because `M_dead` is affine on the whole of
     `[0, 1]` and therefore on each segment.
   - The segment list is worth extracting for its own sake: `level_live` and `level_solve`
     currently state the same three-branch shape twice, in two different forms, and the second
     statement is the one nobody checks against the spec.
   - Spec `docs/specs/t34-videau-spec.md` §9 stops describing the inversion as a bisection and
     states the closed form, with P6's derivation as its reference. §7.1's batched valuation
     becomes moot for the breakpoint half — there is nothing left to interleave — and its
     measurement note should say so.
   - The conventions the bisection had by accident become explicit, because the closed form has
     to choose them: `inf{ p : f(p) ≥ target }`, clamped to `[0, 1]`, a flat segment answering
     with its left bound. They are what sixty steps converged to; written down, they are also
     testable.
   - `testdata/cube_gold.bin` and `testdata/search_cube_gold.bin` are regenerated from the fixed
     C, and `EngineVersion` names the tag.

6. **The port's guard is committed now, so the follow-through is mechanical.**
   `TestClosedFormAgreesWithBisection` holds the proposed closed form against the sixty steps on
   real chains, at 1e-9 in p, on every run. When the upstream tag lands, the port replaces
   `levelSolve`'s body, this test becomes the regression that says the replacement is the same
   function, and the gold files say it is the same function *as the C*.

## Considered options

**Write the closed form here and regenerate the gold.** Rejected on the invariant. See decision
2: the cube gold's current `max|Δ| = 0` is a fact about this port that no local optimisation is
allowed to spend.

**Write the closed form here and keep the bisection as the reference behind a flag.** Rejected
for the reason the cube-efficiency record rejected `CubeXMirror`: a knob on a model gammonNet has
not adopted is a second model with no owner. The measurement does not need it — patching
`levelSolve`, measuring, and reverting produced every number above, and the committed agreement
test reproduces the exactness half with no production surface at all.

**Keep the bisection but run fewer steps.** Rejected: it is neither exact nor bit-identical nor
upstream's. It trades a defensible number for an indefensible one.

**Keep the bisection but skip its early steps analytically, resuming the real loop once the
interval is narrow.** Rejected. It is aimed at bit-identity, and it does not reach it: the
skipped comparisons would be reproduced by comparing `mid` against a *computed* root, and it is
precisely near the root — where rounding in `v0 + n·((p−x0)/d)` decides the comparison — that the
two stop agreeing. A construction whose only merit would be exactness, and which is not exact,
is worse than either end of the trade.

**Do nothing: 35 % of a decision at a score is affordable.** Rejected. It is 35 % of every leaf
of every position of every batch, and the batch is the mode that runs for hours (12.2 pos/s at
2-ply on 16 cores: a 88 000-position database is two hours, and 1.6 of them are this). It is also
the *second* time this file's cost has been attributed to the network and found elsewhere.

## Consequences

- **No behavioural change, no schema change, no `EngineVersion` change today.** Stored analyses
  stay valid; both gold corpora are untouched; the cube gold keeps reporting an exact 0.
- `cube_closedform_measure_test.go` is the instrument and the proposal in one file: the closed
  form itself, an always-on agreement test against the sixty steps, the gap measurement behind
  `BLUNDERDB_MEASURE_CLOSEDFORM`, and the six benchmarks the fiche asked for *before* any change
  — `LevelSolveBisection`, `LevelSolveClosed`, `BuildLevels`, `BuildLevelAnchors`,
  `CubeDecisionAtScore`, `CubeDecisionMoney`, plus `AnalysisBatchThroughput` in pos/s.
- When the upstream tag lands, the port is a body swap plus two regenerated gold files, and the
  expected effect is known in advance: ×1.58 on a 2-ply decision at a score, ×1.28 on the batch,
  4.4e-16 on an equity, and — measured on 669 real decisions — not one move, cost or verdict
  changed.
- Every stored gammonNet analysis will be stale at that tag, jointly with the branch-local
  efficiency. That single sweep is the reason the two travel together.
- The next cost centre after this one is `EvaluateBatch`, at 52.6 % — the network, which is
  where the remaining work is, and where the reproducibility record already decided what may and
  may not be done to it.
