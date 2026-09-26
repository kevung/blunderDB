# The cube's level inversion becomes a closed form, and that is written upstream

Status: accepted.
See also: ADR-0011, ADR-0022, ADR-0029

## Context

`levelSolve` (`engine/gammonnet/cube.go`, upstream `level_solve`) inverts the match cube
model's stake-level curve by sixty steps of bisection. The curve is piecewise linear and
monotone and its pieces are known in advance (ADR-0022), so the inversion is: pick the bracketing
segment, one division. Measured: ×19 on the function; a 2-ply decision at a score 306 → 193 ms
(the cost of money play); batch throughput ×1.28. The result is not bit-identical: worst
|Δ| 4.4e-16 on a valuation, and the cube gold would go from an exact 0 to 1.665e-14. Nothing a
user sees moves on 669 real decisions.

## Decision

1. **The inversion becomes a closed form.** Its gain survives a change of language, so it is a
   conceptual change: written in gammonNet first (ADR-0011 rule 7). By contrast `laneCurve`
   (hoisting segment constants, needed because Go will not inline `levelLive`) is an
   implementation win and stays here.
2. **Nothing in this repository changes before the upstream tag.** The cube gold's exact 0 is a
   fact about this port that no local optimisation may spend.
3. **It is a Configuration change** (ADR-0011 rule 8): not bit-identical, so a new
   `EngineVersion` and every stored gammonNet analysis stale — under ADR-0024 a different last
   bit is a different number in a database.
4. **It ships in one gammonNet tag with the branch-local efficiency** (ADR-0029 rule 4): both
   touch `gn_cube.c`, both regenerate `cube_gold.bin` and `search_cube_gold.bin`, both make
   every stored analysis stale. One release, one port, one stale sweep.
5. **The upstream change**, proposed:
   - `level_solve(level, owner, blend, target)` keeps its signature and loses its loop: it walks
     the level's segments `(x0, y0, x1, y1)` in ascending p — the ones `level_live` selects
     between, dead level included — skips degenerate ones, and returns
     `x0 + (x1 − x0)·(target − v0)/(v1 − v0)` in the bracketing segment. With `blend ≥ 0` the
     endpoints are blended first (exact: `M_dead` is affine on `[0, 1]`).
   - The segment list is extracted once, shared by `level_live` and `level_solve`.
   - Conventions made explicit: `inf{ p : f(p) ≥ target }`, clamped to `[0, 1]`, a flat segment
     answering its left bound.
   - Spec `t34-videau-spec.md` §9 states the closed form instead of a bisection.
6. **The port's guard is committed now**: when the tag lands the port swaps `levelSolve`'s body,
   the agreement test says it is the same function, the gold says it is the C's.

## Consequences

- Until the tag: no behaviour, schema or `EngineVersion` change; the cube gold stays exactly 0.
- Rejected: **the closed form here, gold regenerated** (the gold then measures nothing); **behind
  a flag with the bisection as reference** (a second model with no owner); **fewer bisection
  steps** (neither exact, nor bit-identical, nor upstream's); **skipping early steps
  analytically** (aimed at bit-identity and does not reach it near the root); **doing nothing**
  (35 % of every leaf of every batch, hours on a large database).

## Guard

`TestClosedFormAgreesWithBisection` in
`pkg/blunderdb/engine/gammonnet/cube_closedform_measure_test.go` (always on, 1e-9 in p on real
chains; gap measurement behind `BLUNDERDB_MEASURE_CLOSEDFORM`, plus the benchmarks).
