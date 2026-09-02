# The search values its leaves with the cube

## Status

accepted — decided 2026-09-02. Closes the follow-up ADR-0016 point 7 left open ("`use_cube` is
out of scope, and is the follow-up ticket"). Lands on ADR-0019 (everything that leaves the
engine at a score is normalised equity — unchanged, the conversion is affine and applies to a
cubeful value exactly as it did to a cubeless one) and on ADR-0022 (the live curve the leaves
are now priced with is the one with its tails). Pinned to gammonNet **v1.2.1**, the release
that carries the Crawford dead-value fix found while measuring this; `EngineVersion` names it.

Also fixes a defect found on the way: the pre-roll probability walk (`Searcher.Probs`, the
input of every cube decision the Eval panel shows) reused the root's match state at every
level instead of swapping it, so at 2 ply and an asymmetric score the opponent's replies were
ranked from the wrong side of the score. Exact at 1 ply and at symmetric scores, silently off
at 4-away/2-away. Now threaded like the scalar walk, and held by an identity test.

## Context

The question was put plainly: *"la prise en compte du score dans gammonNet n'est pas complète
ou partielle"* — the opening 6-4 at gammon-go (4-away/2-away) and gammon-save (2-away/4-away)
did not seem to react to the score. Measured on 2026-09-02 (gammonNet
`bench/probe_opening_at_score.py`, 15 opening rolls × 15 score contexts, gnubg 1.08.003 at
2 ply):

| pair | best-move agreement | costs > 0.02 | max |
|---|---:|---:|---:|
| blunderDB (`use_match`, cubeless leaves) vs gnubg **cubeless** | 95 % | 0 | 0.010 |
| blunderDB (`use_match`, cubeless leaves) vs gnubg **cubeful** | 77 % | 27 | 0.064 |
| gnubg cubeless vs gnubg cubeful (control) | 80 % | 23 | 0.071 |
| gammonNet `use_match` + `use_cube` vs gnubg cubeful | 89 % | 2 | 0.035 |

So the score *was* fully taken into account — by the table, exactly as gnubg's own cubeless
evaluation does it (`Utility` with MET gammon prices, `eval.c`), and the port agrees with the C
bit for bit. What was missing is the cube. At 4-away/2-away the trailer's **cubeless** equity is
negative (a loss lands them at 4-away/1-away Crawford) and their gammon price (0.87) sits
*below* the leader's (1.48): cubeless, 24/18 13/9 is right. What makes 8/2 6/2 right is the
double the trailer will turn early — at 2 their gammon wins the match and the leader's gammons
stop mattering. gnubg sees that because its chequer play is cubeful (`Cl2CfMatch`); gammonNet's
`use_cube` sees it (+0.294 for 8/2 6/2 against +0.270); the port, cubeless, could not.

## Decision

1. **Every leaf is valued through the cube model**, at the cube state the position carries,
   at that state's measured efficiency — `SearchConfig.UseCube/CubeOwner/CubeX`, a port of
   `gn_search.c`'s `use_cube`. The owner is mirrored (Owned ↔ Opponent) at every ply *in the
   same calls* that swap the match state; state and owner never travel separately. No
   double/take/pass branches in the tree (spec §8: the reference engines do not either).
   Terminal positions are worth their stake whatever the cube, as in the C.
2. **One configuration, one definition.** `ConfigForPosition` is what gammonNet is asked for a
   position: canonical depth and pruning, the position's referential (ADR-0016), its cube.
   `EvaluatePosition` and internal/gui's pre-roll facts both call it, so a panel can never show
   a fact vector from a differently configured search than the decision beside it.
3. **Facts stay cubeless.** `PreRollFacts.CubelessEquity` and the "avant le jet" line are the
   cubeless equity of the distribution — a fact of the position, by definition — even though
   the search that produced the distribution priced its leaves with the cube.
4. **The probability walk swaps the score.** `probsAt` threads state and owner exactly as
   `positionEquity` does; `TestProbsMatchEquityMatchesPositionEquity` holds the linearity
   identity at 2 ply at 4-away/2-away, where the old walk failed it.
5. **No exact-table shortcut.** Upstream's `node_value` reads exact cubeful equities from a
   shared two-sided table on money bear-off leaves when one is loaded. The port never does —
   this package has no such table — and the gold is generated with none, so the two agree on
   the model path. Documented, not a drift.
6. **A second gold corpus.** `search_cube_corpus.bin` (`GNC2`) carries match and cube states
   per decision; `gold.c` reads both formats; the same gate replays both. The money-cubeless
   gold is byte-identical to before, which is the port's own statement that the cubeless path
   did not move.
7. **`EngineVersion` = "gammonNet v1.2.1"**, and every row stored under an earlier label is
   stale as a whole — the batch job re-runs it. blunderDB has no released build carrying
   gammonNet analyses, so no user database is affected.

## Considered options

**Leave the search cubeless and document it.** Rejected: the panel then disagrees with XG and
gnubg on the most instructive positions at a score — the ones where the cube decides the play —
and a stored analysis would say 24/18 13/9 next to an imported XG analysis saying 8/2 6/2 on the
same board, in the same column.

**Apply the cube only at the root** (rank the shallow sweep cubeful, search deeper cubeless).
Rejected: two valuations in one tree, and the deep pass would overrule the cubeful ordering
with a cubeless one exactly where depth matters.

**Fit a score-specific efficiency first.** Deferred: `x` is the money-measured one (T34), the
same upstream uses at every score. The two remaining outliers of the measurement are both
post-Crawford 2-away/1-away, a model question (the trailer's immediate double is under-priced),
not a port question.

## Consequences

- Checker-move equities at a score are now cubeful normalised equity, the scale an imported
  XG/gnubg analysis sits in; money equities move slightly (cubeful, per unit of cube).
- The Eval panel's cube decision at 2 ply is now computed from a correctly swapped probability
  walk at asymmetric scores.
- `TestIntegrationGate` (BLUNDERDB_GATE) is re-measured under this configuration — its own
  `searcherFor` is now cubeful, like `ConfigForPosition` — and its header records the numbers.
  ADR-0016's note "re-measure once `use_cube` lands" is discharged, with a result worth stating
  plainly: the gate still FAILS by the same two decisions, at the same costs to the fourth
  decimal, and that identity refutes the reason ADR-0016 gave for them. Every decision at that
  score in the fixture is in the **Crawford game**, where there is no cube at all, so `use_cube`
  could never have moved them. What remains to explain them is depth and MET, not a missing
  tranche. Left failing, not loosened.
- Open: the race regime's own search (`internal/gui` `evaluateRaceRegime`) still builds a money
  cubeless walk for its win probability; unchanged here, named as the next candidate for
  `ConfigForPosition`.
