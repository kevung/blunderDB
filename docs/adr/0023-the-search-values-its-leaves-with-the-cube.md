# The search values its leaves with the cube

Status: accepted.
See also: ADR-0016, ADR-0019, ADR-0022, ADR-0029

## Context

With `use_match` alone the search took the score fully into account through the MET, exactly
as gnubg's cubeless evaluation does — but not the cube. At 4-away/2-away the trailer's right
opening play depends on the early double they will turn, which a cubeless leaf cannot see.
Over 15 opening rolls × 15 score contexts against gnubg 2-ply cubeful: cubeless leaves agreed
on 77 % of best moves with 27 costs above 0.02; gammonNet with `use_cube`, 89 % and 2.

## Decision

1. **Every leaf is valued through the cube model** (ADR-0022's curve), at the cube state the
   position carries — `SearchConfig.UseCube/CubeOwner/CubeX`, a port of `gn_search.c`'s
   `use_cube`. The owner is mirrored (Owned ↔ Opponent) at every ply in the same calls that
   swap the match state; state and owner never travel separately. No double/take/pass branches
   in the tree. Terminal positions are worth their stake whatever the cube. Which efficiency a
   leaf reads is ADR-0029's.
2. **One configuration, one definition.** `ConfigForPosition` is what gammonNet is asked for a
   position: canonical depth and pruning, its referential (ADR-0016), its cube.
   `EvaluatePosition`, internal/gui's pre-roll facts and the race regime all call it, so a panel
   never shows facts from a differently configured search than the decision beside them.
3. **Facts stay cubeless.** `PreRollFacts.CubelessEquity` is the cubeless equity of the
   distribution, even though the search that produced it priced its leaves with the cube.
4. **The probability walk swaps the score.** `probsAt` threads state and owner exactly as
   `positionEquity` does.
5. **No exact-table shortcut.** Upstream reads exact cubeful equities from a two-sided table on
   money bear-off leaves when one is loaded; the port never does, and the gold is generated
   without one, so both agree on the model path.
6. **Two gold corpora**: the money-cubeless one and `search_cube_corpus.bin` (`GNC2`), carrying
   match and cube states per decision; the same gate replays both.
7. Version label — see ADR-0011 rule 8.

## Consequences

- Checker-move equities at a score are cubeful normalised equity, the scale of an imported
  XG/gnubg analysis; money equities are cubeful per unit of cube.
- The integration gate's two red decisions (ADR-0014) are in the Crawford game, where there is
  no cube: `use_cube` could not move them, and did not.
- Rejected: **a cubeless search, documented** (disagrees with XG and gnubg exactly where the cube
  decides the play); **the cube at the root only** (two valuations in one tree, the deep pass
  overruling the cubeful ordering); **a score-specific efficiency first** (a model question for
  gammonNet, not a port question).

## Guard

`pkg/blunderdb/engine/gammonnet/search_cube_test.go`
(`TestProbsMatchEquityMatchesPositionEquity`, at 2-ply, 4-away/2-away), `gold_corpus_test.go`
against `testdata/search_cube_gold.bin`.
