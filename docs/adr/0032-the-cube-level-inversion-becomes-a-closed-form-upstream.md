# The cube's level inversion becomes a closed form, and that is written upstream

Status: accepted.
See also: ADR-0011, ADR-0022, ADR-0029

## Context

`levelSolve` (`engine/gammonnet/cube.go`, upstream `level_solve`) inverts the match cube
model's stake-level curve. The curve is piecewise linear and monotone and its pieces are known
in advance (ADR-0022), so the inversion is: pick the bracketing segment, one division. Against
sixty steps of bisection, measured: ×19 on the function; a 2-ply decision at a score 306 → 193 ms
(the cost of money play); batch throughput ×1.28. The two are not bit-identical: worst
|Δ| 4.4e-16 on a valuation, and the cube gold of a locally rewritten port would move from an
exact 0 to 1.665e-14. Nothing a user sees moves on 669 real decisions.

## Decision

1. **The inversion is a closed form.** Its gain survives a change of language, so it is a
   conceptual change: written in gammonNet first (ADR-0011 rule 7); gammonNet's `level_solve`
   (spec §9) holds it and the port follows.
2. **The port changes only with the upstream tag.** The cube gold's exact 0 is a fact about
   this port that no local optimisation may spend.
3. **It is a Configuration change** (ADR-0011 rule 8): not bit-identical, so a new
   `EngineVersion` and every stored gammonNet analysis stale — under ADR-0024 a different last
   bit is a different number in a database.
4. **It stands on its own.** No branch-local efficiency accompanies it (ADR-0029 rule 4); the
   closed form alone determines `cube_gold.bin` and `search_cube_gold.bin` and the staleness of
   stored analyses.
5. **The upstream form**:
   - `level_solve(level, owner, blend, target)` has no loop: it walks
     the level's segments `(x0, y0, x1, y1)` in ascending p — the ones `level_live` selects
     between, dead level included — skips degenerate ones, and returns
     `x0 + (x1 − x0)·(target − v0)/(v1 − v0)` in the bracketing segment. With `blend ≥ 0` the
     endpoints are blended first (exact: `M_dead` is affine on `[0, 1]`).
   - The segment list is extracted once, shared by `level_live` and `level_solve`.
   - Conventions made explicit: `inf{ p : f(p) ≥ target }`, clamped to `[0, 1]`, a flat segment
     answering its left bound.
   - Spec `t34-videau-spec.md` §9 states the closed form.
6. **The port's guard**: the agreement test says `levelSolve` is the function a converging
   bisection computes, the gold says it is the C's.

## Consequences

- The closed form carries its own `EngineVersion`; the cube gold is exactly 0 against the C's
  closed form.
- Rejected: **the closed form here, gold regenerated** (the gold then measures nothing); **behind
  a flag with the bisection as reference** (a second model with no owner); **fewer bisection
  steps** (neither exact, nor bit-identical, nor upstream's); **skipping early steps
  analytically** (aimed at bit-identity and does not reach it near the root); **doing nothing**
  (35 % of every leaf of every batch, hours on a large database).

## Guard

`TestClosedFormAgreesWithBisection` in
`pkg/blunderdb/engine/gammonnet/cube_closedform_measure_test.go` (always on, 1e-9 in p on real
chains, against the sixty-step bisection kept there as the reference, plus the benchmarks);
`TestCubeDecideMatchesTheGoldFile` (bit-exact against the C).
