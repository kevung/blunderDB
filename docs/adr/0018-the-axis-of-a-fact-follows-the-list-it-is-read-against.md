# The axis of a fact follows the list it is read against

## Status

accepted — decided 2026-08-31, one day after ADR-0017 and before any of it had been used in
anger. **Amends ADR-0017** rules 1, 2 (its Défi corollary) and 5; supersedes nothing. ADR-0017's
partition — *a quantity is either a fact of the position or part of a decision* — is untouched and
is the reason this decision is small: only the **axis** and the **skin** change, never what is
computed, stored, or which question the board is asking.

Depends on nothing new. ADR-0009, ADR-0010, ADR-0012 and ADR-0016 are all load-bearing below but
none of them moves.

## Context

ADR-0017 shipped in `f01022ea` and was looked at. Three things it did not predict:

**The panel is mostly empty on a checker position.** With dice on the board, `showRaceDecision`
and `showGenericCube` are both false and the `{:else if !hasDiceSet}` placeholder does not apply,
so `.top-row` holds exactly two children: the facts table (intrinsic width, about a third of the
panel) and `.badges-col`, which carries `margin-left: auto`. The row was drawn for three blocks;
with two, that `auto` pushes the badges to the far edge and manufactures a band of white across
the middle. Below it, `.checker-table` is `width: 100%` and fully ruled. The eye reads a small
floating table, a void, and an unrelated grid.

**Two visual idioms collide, in both panels.** `PositionFactsTable` is borderless — hairline
`#eee` row separators, small grey uppercase headers, a blue lead value, `tabular-nums`.
`CubeVerdictTable` and `CandidateMovesTable` are the pre-ADR-0008 idiom — a full `1px solid #ddd`
grid, `#f2f2f2` header cells, `<th>` inside `<tbody>`. ADR-0017 moved quantities *between* these
components without ever unifying them, and since both panels mount all three
(`AnalysisPanel.svelte:434-441`), Analysis inherited the same collision. A full grid on eleven
columns is what makes a dense list unreadable: every cell is a closed rectangle, so the eye cannot
run down a column.

**The facts table is the transpose of one row of the list it sits above.** This is the finding
that decided the shape of the answer, and it is a fact about the engine, not a matter of taste:

- `domaineval.go:212` — every candidate carries `mine := InvertProbs(&c.Probs)`, so
  `PlayerWinChance … OpponentBackgammonChance` are **relative to the player on roll**.
- `gammonnet_eval.go:216-224` — `preRollFacts` builds the same six quantities **in the same
  frame**.
- `cube.go:381-383` — `CubelessValue` *is* `valueFromProbs`, the very function `search.go:378`
  uses for each candidate's `Equity`. Same scale, same Referential (ADR-0016).

So the pre-roll vector and every move row are the same seven numbers, and ADR-0017 rendered the
baseline in one axis (bottom/top/Δ) and the list it serves in the other (player/opponent), in a
second visual idiom, on the other side of a void. A baseline in a different axis from the table it
serves is not a baseline; it is a second table. That is precisely what it looked like.

ADR-0017 also asserted that Défi "regains a sound definition: three zones … and nothing escapes
the mask". That was not true when it was written. `maskedDecision` guards only `.decision-race`
and `.decision-cube`, both gated on `!hasDiceSet`, while `CandidateMovesTable` is mounted with no
mask at all (`EPCPanel.svelte:409`) — so with dice set the whole ranked list, equities and per-move
probabilities included, has always been shown in clear.

## Decision

**A per-side quantity is read in rows; a quantity that has a list of options to be read against
takes that list's axis.**

1. **The race block is per side, always.** EPC, pip count, wastage, mean rolls and their
   dispersion are properties of one player's home board. They keep the `bottom` / `top` / `Δ`
   rows, with dice and without, in both panels. They never migrate.
2. **The pre-roll vector takes the axis of the Decision it heads.** Dice on the board → it is
   rendered at the trait, as a band inside the candidate table itself, pinned under the sticky
   header and never scrolled, in the columns
   `— | cubeless equity | (empty) | PW | PG | PB | OW | OG | OB`. No dice → there is no list
   carrying those columns (`CubeVerdictTable` has three: action, equity, error), so it is read per
   side exactly as ADR-0017 has it, Δ row included.
3. **The band is a reference mark, never the head of the ranking.** It is labelled *before the
   roll*, set in italic on a neutral ground, closed by a heavy rule, and its **error cell is empty,
   not `0.000`**. The gap between it and any move row contains the value of the roll — the luck of
   the position (ADR-0010) — and must never read as the merit of the play.
4. **Provenance leaves the rows.** Depth and engine are constant across every row of a live
   evaluation (`domaineval.go:206-207` sets one `depthLabel` and one `EngineVersion` for all
   candidates), so in Eval they collapse into a single strip carrying regime badge, depth, engine
   link and the Défi toggle. In Analysis they stay per row — `sortedMoves` is sorted *by engine*
   (`AnalysisPanel.svelte:406-413`), where the columns carry real information. This is a prop, not
   a deletion.
5. **One idiom for all three tables.** `CandidateMovesTable` and `CubeVerdictTable` adopt
   `PositionFactsTable`'s: no cell grid, hairline horizontal separators, small grey uppercase
   headers on a transparent ground, `tabular-nums`, and a heavier `2px` vertical rule only at group
   boundaries (move | equities | player | opponent). Both panels change together, by construction —
   ADR-0017 rule 5 already made that unavoidable. This is ADR-0008's "hierarchy comes from weight
   and colour" applied to the last components that predate it. Scope stops at these three: no table
   token is introduced in `style.css` for the rest of the app in this pass.
6. **Défi has three zones, and what they cover follows the board.** No dice: `bottom`, `top` (the
   per-side vector) and `decision` (the cube block) — unchanged. Dice on a race: `bottom`, `top`
   (the race block) and `decision` = **the pre-roll band and the candidate list together**, behind
   one `···`, the pattern `.decision-cube-masked` already uses. Dice off a race: the `decision`
   zone alone. Masking a sorted list's values while showing its rows would reveal nothing, because
   the order *is* the answer: it is the whole block or none of it.

**The layout never manufactures a void.** `margin-left: auto` on `.badges-col` is replaced by
`justify-content: space-between` on `.top-row`: with one child it sits flush left, with two it
spreads. The void was a rule producing it, not a size to be tuned.

## Considered options

- **Repaint only, keep ADR-0017's structure.** Cheapest, and it addresses the idiom collision. It
  leaves the void, and leaves the baseline in the wrong axis — which is the part that makes the two
  halves read as unrelated.
- **Unify on the ruled idiom instead** (grid up `PositionFactsTable`). Analysis would not change
  and the diffs would be tiny. Rejected: it makes the facts table worse — thirty new borders across
  three rows and nine columns for no information — and walks back ADR-0008.
- **Keep the vector per side always** (ADR-0017 as written). This is the status quo the complaint
  is about.
- **Render the vector at the trait always**, cube positions included. One axis, no exception, the
  purest rule. Rejected: it costs the Δ row in the one position where the gap between the two camps
  *is* the decision.
- **Put the race columns into the band too**, so everything lives in one object. Rejected: EPC,
  wastage and σ are per side by construction; folding two axes into one row is unreadable the
  moment more than one race column is wanted.
- **Four Défi zones** (`bottom`, `top`, `pre-roll`, `list`). Rejected: four clicks to open a race
  position, and a pre-roll band alone is not an exercise anybody can attempt.

## Consequences

- **Two of ADR-0017's own claims become false and are corrected here rather than left standing.**
  Its rule 1 promised "the table does not move, five columns arrive"; the truth after this decision
  is the reverse — on a checker position five race columns stay and four probability columns leave.
  And its Défi corollary claimed nothing escaped the mask; the list always did, and rule 6 above is
  what actually closes it.
- **This is the second axis decision on the same panel in two days, and the last one that is
  cheap.** ADR-0017 rejected exactly this change ("the same quantity would change place depending
  on the position, which is the opposite of a smooth transition") and it was not wrong to: what it
  could not know was the price. The void was paid for the promise and turned out to be two thirds
  of the panel's width. The change of axis here is not arbitrary — it is *caused by the list
  appearing*, so the user provokes it by placing dice and watches it happen. A third reversal would
  cost the panel its credibility; this one is recorded so the next reader argues with the reasons
  rather than rediscovering them.
- **`PositionFactsTable` splits by usage, not by panel.** Its race columns serve the top row in
  both panels; its probability rows serve the no-dice case only. Its `preRoll` prop goes away — the
  case it flagged is now rendered somewhere else entirely.
- **The escalation gets quieter.** With provenance out of the rows, moving from 0-ply to the
  display depth changes one label instead of rewriting a whole column of N rows. ADR-0017 rule 3
  already wanted this ("only the depth label changes") without giving itself the means.
- **The candidate list drops from eleven columns to nine in Eval**, and the cube block loses the
  `info-table` outright. Neither loses a number the panel does not still show.
- **ADR-0012's requirement that the regime be named on screen is met by the strip**, which is why
  the strip stays at the top rather than becoming a footer: the badge qualifies the numbers it sits
  above, and the Défi toggle is a control, not metadata.
- **The no-scroll Playwright assertion ADR-0017 left as a follow-up is still owed**, and is now
  worth more: with the band pinned inside the scroll region, "only the list scrolls" has a second
  way to break.
- **Nothing in `pkg/` changes.** No engine, no regime, no stored row, no wire type. `preRollFacts`
  keeps paying its measured +36 % on the one position that pays it, for the same reason ADR-0017
  bought it: the band is the thing that reading a candidate list against a baseline requires.
