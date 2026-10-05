# Cube efficiency is measured per cube state, and read at the root

Status: accepted.
See also: ADR-0011, ADR-0022, ADR-0023, ADR-0032

## Context

`DefaultEfficiency` (`engine/gammonnet/cube.go`) returns one coefficient per cube state — owned
**0.566**, centred **0.688**, opponent **0.687** — each fitted by least squares against a
different column of gammonNet's exact two-sided table (spec §3: never borrowed from another
engine). The search fixes `CubeX` at the root while mirroring the owner at every ply, and
`Decide` prices `eDT` at the current owner's coefficient; the C (`gn_search.c`, `gn_cube.c`)
does exactly the same. Pricing each branch at its own coefficient instead was measured on 669
real decisions (55 % with a turned cube): a mean 0.005 normalised equity per leaf, 0 of 604
cube verdicts flipped, 0 of 60 best moves changed.

## Decision

1. **The cube efficiency stays indexed by cube state, and measured.** blunderDB does not adopt
   gnubg's position-class efficiency (contact 0.68, bearoff 0.6, race interpolated). The
   divergence from gnubg and XG is deliberate.
2. **The coefficient is read at the root.** A mirrored leaf is priced with the root's
   coefficient and `eDT` with the current owner's: that is the model, upstream's and this
   port's. A branch-local coefficient buys a few thousandths of equity and no verdict or move,
   which does not pay for a second model.
3. **The port follows the C** (ADR-0011 rule 7): a local "fix" of either reading is a port
   divergence that turns the cube gold red.
4. **No per-owner coefficient is proposed upstream.** The mirrored efficiency (`cube_x[3]`
   indexed by owner, `e_dt` at the opponent's coefficient) is void. Reopening it takes a new
   upstream measurement in which a verdict or a move changes, and a gammonNet tag — never a
   change here first.
5. **The reading is commented where a reader would otherwise "fix" it** — at
   `SearchConfig.CubeX`, `DefaultEfficiency` and `Decide`'s `eDT` line — each pointing here.

## Consequences

- No behaviour, schema or `EngineVersion` change follows from this record; both gold corpora
  hold the reading exactly.
- Rejected: **branch-local efficiency** (0.005 per leaf, nothing a user sees); **correcting the
  port and regenerating the gold** (the gold then measures nothing); **gnubg's position-class
  efficiency** (forbidden by spec §3, read off a manual, no classifier here); **re-fitting
  locally against blunderDB's own table** (a separate question from which coefficient a leaf
  reads); **a `CubeXMirror` flag** (a second model with no owner).

## Guard

`pkg/blunderdb/engine/gammonnet/cube_efficiency_measure_test.go` (`BLUNDERDB_MEASURE_CUBEX`,
and the gate depth replay behind `BLUNDERDB_MEASURE_GATE_DEPTH`); `cube_gold_test.go`.
