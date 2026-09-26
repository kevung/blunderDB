# The axis of a fact follows the list it is read against

Status: accepted.
See also: ADR-0017, ADR-0020, ADR-0021, ADR-0008

## Context

The pre-roll vector and every candidate-move row are the same quantities in the same frame:
candidates carry probabilities relative to the player on roll, `preRollFacts` builds the same
six in that frame, and `CubelessValue` is the `valueFromProbs` each candidate's equity uses.
Rendering the baseline per side (bottom/top/Δ) above a list read per option (player/opponent)
made it a second table, not a baseline. The tables also used two visual idioms (hairline vs full
cell grid), and with dice set the candidate list escaped the Défi mask entirely.

## Decision

**A per-side quantity is read in rows; a quantity that has a list of options to be read against
takes that list's axis.**

1. **The race block is per side, always.** EPC, pip count, wastage, mean rolls and their
   dispersion keep the `bottom` / `top` / `Δ` rows, with dice and without, in both panels.
2. **The pre-roll vector takes the axis of the Decision it heads.** Dice on the board → a band
   inside the candidate table, pinned under the sticky header, in the columns
   `— | cubeless equity | (empty) | PW | PG | PB | OW | OG | OB`. No dice → no list carries those
   columns, so it is read per side, Δ row included.
3. **The band is a reference mark, never the head of the ranking.** Labelled *before the roll*,
   italic on a neutral ground, closed by a heavy rule, and its error cell is **empty, not
   `0.000`**: the gap to any move row is the luck of the roll (ADR-0010), never the merit of the
   play.
4. **Provenance leaves the rows.** Depth and engine are constant across a live evaluation, so in
   Eval they collapse into one strip with the regime badge, engine link and Défi toggle (its
   placement: ADR-0020 rule 8). In Analysis they stay per row — moves are sorted by engine there,
   so the columns carry information. A prop, not a deletion.
5. **One idiom for all the Eval/Analysis tables.** No cell grid, hairline horizontal separators,
   small grey uppercase headers on a transparent ground, `tabular-nums`, and a `2px` vertical
   rule only at group boundaries (move | equities | player | opponent). This is ADR-0008 rule 2
   applied; no table token is added to `style.css` for the rest of the app.
6. **Défi has three zones, and what they cover follows the board.** No dice: `bottom`, `top`
   (per-side vector), `decision` (cube block). Dice on a race: `bottom`, `top` (race block), and
   `decision` = the pre-roll band **and** the candidate list together behind one `···`. Dice off
   a race: the `decision` zone alone. A sorted list is masked whole or not at all — its order is
   the answer. How the cube block masks: ADR-0020 rule 7.

## Consequences

- The axis change is caused by the list appearing: the user provokes it by placing dice.
- `PositionFactsTable` splits by usage: race facts in both panels, probability rows only when
  there are no dice.
- The escalation changes one label, not a column of N rows (ADR-0017 rule 3).
- The Eval candidate list has nine columns; nothing in `pkg/` depends on this decision.
- Rejected: repaint only (leaves the baseline in the wrong axis). Rejected: the ruled idiom
  everywhere (adds borders carrying no information, walks back ADR-0008). Rejected: the vector
  at the trait always (loses the Δ row where the gap between camps is the cube decision).
  Rejected: race columns in the band (per-side by construction). Rejected: four Défi zones
  (a lone pre-roll band is not an exercise).

## Guard

`frontend/tests/e2e/eval-panel-no-scroll.spec.js`;
`frontend/src/__tests__/CubeVerdictTable.challengeMask.test.js`.
