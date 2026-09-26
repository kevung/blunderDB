# Cube efficiency is measured per cube state, and read at the root until gammonNet says otherwise

Status: accepted.
See also: ADR-0011, ADR-0022, ADR-0023, ADR-0032

## Context

`DefaultEfficiency` (`engine/gammonnet/cube.go`) returns one coefficient per cube state — owned
**0.566**, centred **0.688**, opponent **0.687** — each fitted by least squares against a
different column of gammonNet's exact two-sided table (spec §3: never borrowed from another
engine). The search fixes `CubeX` at the root while mirroring the owner at every ply, and
`Decide` prices `eDT` at the current owner's coefficient; the C (`gn_search.c`, `gn_cube.c`)
does exactly the same. On 669 real decisions (55 % with a turned cube) the mis-read costs a
mean 0.005 normalised equity per leaf, flips 0 of 604 cube verdicts and changes 0 of 60 best
moves.

## Decision

1. **The cube efficiency stays indexed by cube state, and measured.** blunderDB does not adopt
   gnubg's position-class efficiency (contact 0.68, bearoff 0.6, race interpolated). The
   divergence from gnubg and XG is deliberate.
2. **The coefficient belongs to the branch the curve is on**; reading the root's at a mirrored
   leaf, or the owner's for `eDT`, is a small defect of the model's statement. Upstream's exact
   path (`gn_bearoff_equities`, indexed by the local owner) already agrees.
3. **The correction is gammonNet's to write** (ADR-0011 rule 7). Until its tag lands, the port
   keeps reading the root's coefficient — "fixing" it here is a port divergence that turns the
   cube gold red.
4. **The upstream change**, proposed:
   - `GnSearchConfig.cube_x` becomes `double cube_x[3]`, indexed by `GnCubeOwner`;
     `gn_search_use_cube(config, owner, const double x[3])`. The same value in all three slots
     reproduces today's behaviour bit for bit, so the gold stays a control.
   - `node_value` and the batch path index it by the **local** owner they already pass.
   - `gn_cube_decide` prices `e_nd` at `x[owner]`, and `e_dt` plus the take-point `level_solve`
     at `x[GN_CUBE_OPPONENT]`.
   - Spec §4 becomes `E_nd = c · E(x_courant; état courant)`,
     `E_dt = 2c · E(x_adverse; adversaire possède)`; §8 step 2: a leaf takes the coefficient of
     its local cube state.
   - It ships in one tag with ADR-0032's closed form (ADR-0032 rule 4).
5. **The divergence is commented where a reader would otherwise "fix" it** — at
   `SearchConfig.CubeX`, `DefaultEfficiency` and `Decide`'s `eDT` line — each pointing here.

## Consequences

- No behaviour, schema or `EngineVersion` change until upstream moves; then both gold corpora
  are regenerated from the C and every stored gammonNet analysis is stale (ADR-0011 rule 8).
  Expected effect: a few thousandths of equity on 55 % of positions, no move or verdict.
- Rejected: **correcting the port and regenerating the gold** (the gold then measures nothing);
  **gnubg's position-class efficiency** (forbidden by spec §3, read off a manual, no classifier
  here); **re-fitting locally against blunderDB's own table** (a separate question from which
  coefficient a leaf reads); **a `CubeXMirror` flag** (a knob on a model gammonNet has not
  adopted is a second model with no owner).

## Guard

`pkg/blunderdb/engine/gammonnet/cube_efficiency_measure_test.go` (`BLUNDERDB_MEASURE_CUBEX`,
and the gate depth replay behind `BLUNDERDB_MEASURE_GATE_DEPTH`); `cube_gold_test.go`.
