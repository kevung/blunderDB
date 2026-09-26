# The referential is a property of the position, and the search honours it

Status: accepted.
See also: ADR-0011, ADR-0013, ADR-0019, ADR-0023

## Context

A cubeless-money search ranks moves identically at every score: the opening 6-4 came back bit
for bit the same at money, 2-away/4-away and double match point, although a gammon is worth
most of the game in the first and nothing in the last. The C reference already implements the
match-aware search (`use_match`); the gap was a porting gap. blunderDB's MET is the same
Kazaross-XG2 table as the C's (float32 vs double: ~1e-7 on the MWC), and blunderDB carries
Crawford inside the away score (`-1` money, `0` 1-away post-Crawford, `1` 1-away Crawford game).

## Decision

**A referential (money or match) is a property of the position, selected by its away score,
and every number blunderDB computes or stores about that position is in it.**

1. **The Go search ports `use_match`**, following `gn_search.c`: the match state travels in
   `SearchConfig`, is swapped at every ply exactly where the value is negated, and terminal
   nodes are valued through the MET. Inside the search nodes are on the antisymmetric
   `2×MWC−1` scale; what leaves the engine is ADR-0019's.
2. **The C is the oracle for structure, blunderDB's MET for the numbers.** Parity at 1e-6 on
   the equity, exact on the chosen play (bar exact ties).
3. **One translation from position to `MatchState`** (`MatchStateFromPosition` /
   `MatchStateFromScores`). It decodes both Crawford sentinels (`0 → {Away 1, post-Crawford}`,
   `1 → {Away 1, Crawford}`) and, beyond the MET's horizon, **refuses** — never a silent
   fallback to money.
4. **The batch job's one exception to ADR-0013**: it may rewrite an analysis if and only if
   that analysis carries an older gammonNet label (ADR-0011 rule 8). XG, GNUbg and BGBlitz
   analyses stay untouchable.
5. **`Searcher.Probs` is match-aware too**, since it walks the tree with the same valuation.
   The race panel's verdict type is `race.CubeVerdict`, all of its fields in the referential.
   The race regime builds its search through `ConfigForPosition` (ADR-0023 rule 2).
6. **The equity column states its referential** (`analysis.equityMoney` /
   `analysis.equityMatch`), decided by one predicate, `isMoneyPosition`
   (`frontend/src/utils/cubeDecision.js`), and documented in `doc/source/manuel.rst`. **No
   setting**: a global toggle would make two analyses of one position incomparable depending
   on a checkbox at batch time.
7. **Leaves valued with the cube** — see ADR-0023 rule 1.

## Consequences

- Displayed numbers at a match score are on a different scale from money and gammon-heavy plays
  move in the ranking — correct, and why the header states the referential.
- Cube-panel numbers move at match scores even with `Decide` untouched, because the pre-roll
  distribution comes from a match-aware tree; `cube_gold_test.go` feeds fixed distributions
  and is unaffected.
- A match valuation is six MET lookups, invisible next to a forward pass.
- Jacoby stays invisible to the search, as in the C: it is a cube-decision rule. Money play
  with Jacoby keeps over-valuing gammons in move choice.
- Rejected: **re-porting `gn_met.c`** (two METs, cube verdicts changed, for ~1e-7); **judging by
  XG alone without C gold** (a sign error can pass a cost criterion by luck); **bumping the
  version without recomputing** (a column silently mixing referentials); **a separate match
  column** (schema bump, and imported XG analyses already sit on the match scale in the same
  column).

## Guard

`search_test.go` / `gold_test.go` (C parity), `pkg/blunderdb/engine/gammonnet/integration_gate_test.go`,
`frontend/src/__tests__/analysisRows.test.js`.
