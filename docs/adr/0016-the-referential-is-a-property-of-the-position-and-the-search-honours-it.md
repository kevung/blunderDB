# The referential is a property of the position, and the search honours it

## Status

proposed — 2026-08-31

## Context

blunderDB's gammonNet search values every node as **cubeless money equity**, at every score.
`search.go` says so in its own header ("The search is CUBELESS and MONEY-only … it arrives
with the MET work"), and the effect is not subtle. Measured on the opening 6-4 at 2-ply,
across money, 7-away/7-away, 4-away/2-away, 2-away/4-away, 2-away/2-away and 1-away/1-away,
the five candidates come back **bit for bit identical**:

```
13/9 24/18   eq=+0.01031   W=0.5045 G=0.1352
6/2 8/2      eq=+0.00950   W=0.4993 G=0.1549
18/14 24/18  eq=+0.00211   W=0.5071 G=0.1214
```

`6/2 8/2` is the gammonish play — two extra points of gammon chance over the leader. At
2-away/4-away those points are most of what the game is about; at double match point they are
worth precisely nothing. The engine prices them the same in both, because `pos.Score` never
reaches the search. It reaches exactly one function, `evaluateCube`, and only to build a
`MatchState` for the cube decision.

This is a **porting gap, not an engine bug**. The C reference already implements the whole
mechanism — `gn_search_config_match`, `use_match`, the match state swapped at every ply,
`gn_match_equity`, a MET-valued `terminal_value` — and blunderDB's Go tranche deliberately
deferred it. There is nothing to fix upstream and nothing to propagate: the work is to port
what is already there.

The gap is not hidden, either. `integration_gate_test.go` already excludes 32 decisions from
its cost criterion, with the reason spelled out: our equities are money, XG's are match, "the
two scales coincide away from match point but diverge sharply once either side is 1-away".
ADR-0014 lists "a silently ignored score" among the botched-port symptoms the gate exists to
catch — and the gate is currently looking away from the one it has.

Two facts shaped the decision.

**The two MET tables are the same table.** blunderDB routes match equity through
`engine.GnuBGGetME` rather than a re-ported `gn_met.c` (ticket #122), which looks like it
forecloses parity with the C. It does not. `gn_met_table.h` is Kazaross-XG2 read from
`Kazaross-XG2.xml`; `engine/met.go` computes Zadeh and then **overlays** the same
Kazaross-XG2 values (`overlayKazarossXG2`, 25×25 pre-Crawford, 24 post-Crawford). The C's own
header records the cross-check: "the 625 pre-Crawford entries agree exactly". The two
implementations differ only at post-Crawford 25-away, and beyond 25-away where the C refuses
and blunderDB extends with Zadeh to 64. The C tables are `double` and blunderDB's are
`float32`, so parity is ~1e-7 on the MWC rather than bit-exact — orders of magnitude below
anything that changes a move choice.

**blunderDB carries Crawford inside the away score.** `-1` is money, `0` is *1-away,
post-Crawford*, `1` is *1-away, Crawford game*, `n ≥ 2` is n-away (`domain.go:22`,
`Board.svelte:1530`, the remap at `parser.go:134`). No code decodes the `0` sentinel:
`domaineval.go:180` passes `pos.Score[mover]` straight into `MatchState.AwayOnRoll`, and
`MatchState.IsValid` rejects anything below 1. Every post-Crawford cube decision at match
point is therefore **already refused today** — "cube decision not evaluable at this score" —
and a match-aware search would inherit the same hole on every checker decision.

## Decision

**A referential (money or match) is a property of the position, selected by its away score,
and every number blunderDB computes or stores about that position is in it.**

1. **Port `use_match` into the Go search**, following `gn_search.c` structurally: the match
   state travels in `SearchConfig`, is swapped at every ply exactly where the value is
   negated, and terminal nodes are valued through the MET rather than by
   `terminalEquity`. Nodes are valued as `2 × MWC − 1`, which is antisymmetric under the swap
   and is why the C works on that scale rather than on raw MWC.

2. **The oracle is the C for structure, blunderDB's MET for the numbers.** Parity is asserted
   at 1e-6 on the equity and exactly on the chosen play (bar exact ties, cf. `sortByEquity`).

3. **One translation from position to `MatchState`**, replacing the three near-copies at
   `domaineval.go:175`, `gammonnet_eval.go:200` and `integration_gate_test.go:293`. It decodes
   both sentinels (`0 → {Away: 1, Crawford: false}`, `1 → {Away: 1, Crawford: true}`), which
   fixes post-Crawford cube decisions on the way past. Beyond the MET's horizon it **refuses**
   — never a silent fallback to money, which is the very bug being closed.

4. **`EngineVersion` becomes `gammonNet v1.1.0`.** ADR-0011 tied the label to a weights bump;
   a change of valuation semantics is at least as structural, and the label is the only handle
   for finding stale rows later. The batch job gains a narrow exception to ADR-0013: it may
   rewrite an analysis **if and only if** that analysis carries an older gammonNet label. XG,
   GNUbg and BGBlitz analyses stay untouchable — ADR-0013 protects the *imported* analysis, not
   our own.

5. **`Searcher.Probs` becomes match-aware too**, since it walks the tree with the same
   valuation (the C says so explicitly). The race panel's `race.Money` therefore stops being
   money: `Cubeless` follows the referential like its three neighbours, and the type is renamed
   to something honest (`race.CubeVerdict`). It already mixed scales — `Cubeless` in points
   next to `NoDouble`/`DoubleTake`/`DoublePass` in MWC — and this makes the mixture impossible
   to keep.

6. **The equity column states its referential** ("Équité (money)" / "Équité (match)"), with the
   change documented in `doc/source/manuel.rst`. **No setting**: a referential is a property of
   the position. A global toggle would make two analyses of the same position incomparable
   depending on the state of a checkbox at the time the batch ran.

7. **`use_cube` is out of scope**, and is the follow-up ticket. One tranche, one semantic
   change: `use_cube` mirrors `cube_owner` at every ply on top of the match-state swap, and two
   simultaneous sign conventions in one recursion is exactly the failure mode `search.go`'s
   header warns about — plausible output, no crash, no warning.

## Considered options

**Re-port `gn_met.c` for bit-exact parity.** Rejected: it puts two MET implementations in
blunderDB, or forces the cube decision back onto `gn_met.c` and reverses #122, changing cube
verdicts already shipped. The tables coincide over the whole practical domain anyway, so
bit-exactness buys ~1e-7.

**Judge by XG alone, no C gold.** Rejected: a sign error on the match-state swap can pass a
32-decision cost criterion by luck. A gold file sees it on the first row.

**Bump the version but never recompute.** Rejected: it leaves a database silently mixing money
and match analyses in one column, indefinitely.

**Keep money in `Equity`, add a match column.** Rejected: a schema bump and a triple-sync for a
field the default sort would still ignore — and the imported XG analyses sitting in the same
column are already on the match scale.

## Consequences

- **Displayed numbers change at every match score.** They get smaller in absolute terms
  (normalised equity spans [−1, +1], money spans [−3, +3]) and gammon-heavy plays move in the
  ranking. This is the correct behaviour and matches XG and gnubg; it will still look like a
  regression to anyone who does not read the column header, which is why the header changes.
- **The gate's 1-away exclusion is deleted.** Those 32 decisions must clear the 0.05 cost
  block. That is the acceptance criterion, and it is a test that fails today.
- **Cube panel numbers move at match scores** even though `Decide` is untouched, because the
  pre-roll distribution now comes from a match-aware tree. `cube_gold_test.go` feeds fixed
  distributions to `Decide` and stays green — which is correct, not a gap.
- **Cost is negligible.** `GnuBGGetME` is a table lookup; a match valuation is six of them
  against a five-float dot product, both invisible next to a 196→512→512→256→128→5 forward
  pass.
- **Jacoby remains invisible to the search**, in Go as in C — it is a cube-decision rule in
  both. Money play with Jacoby therefore keeps over-valuing gammons in move choice. Deliberate,
  upstream's choice, and not this ADR's subject.
