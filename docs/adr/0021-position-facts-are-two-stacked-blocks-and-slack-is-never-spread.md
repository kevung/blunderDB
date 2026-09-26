# Position facts are two stacked blocks on one grid, and slack is never spread between blocks

Status: accepted.
See also: ADR-0017, ADR-0020

## Context

At the default window (1024 px, 996 px of Eval content width), a facts table of ten columns
beside the cube block overflowed in seven languages of nine (up to 1125 px in Greek), so the cube
block wrapped under the facts and the content row grew from 108 px to 201 px in a 250 px panel.
In the two languages that fit, a redouble label or a three-digit pip count tipped it over.
Compacting cells still left four languages over budget; a fix that must be re-measured per
language is not a fix. Separately, the Analysis panel spread its slack between its blocks.

## Decision

1. **`PositionFactsTable` renders two stacked blocks, not one line of ten columns.** The
   probability block (win, gammon, backgammon, cubeless) and the race block (EPC, pip count,
   wastage, mean rolls, standard deviation) are two `<tbody>`s, each with its own header row
   and its own bottom / top / Δ rows.
2. **The two blocks share one column grid** — two bodies of one table, never two tables — so
   they share left and right edges, column stops and one column of side markers. The grid is as
   wide as the wider block; the probability block ends in an empty cell. Blocks are separated by
   air, not a border.
3. **Height is spent here because it is free here.** A cube decision exists only without dice
   and the candidate list only with dice (ADR-0017 rule 2), so they never compete. With dice,
   only the race block is rendered and the list keeps its height.
4. **Blocks are laid out at a constant gap; leftover width is left over.** No panel distributes
   slack between blocks — no `space-between`, no `margin-left: auto` on a content block. Slack
   accumulates at the end of the row. Eval and Analysis both use a 20 px gap.
5. **Wrapping stays as a fallback only.** `.top-row` keeps `flex-wrap`, so a hand-narrowed panel
   stacks rather than clips; the default window does not wrap in any language (widest case needs
   824 px of 996).

## Consequences

- The cube decision sits beside the facts at the default window in all nine languages, with
  172–322 px of margin; a translation or a pip count no longer changes the layout. The content
  row is 178 px in every language.
- Side markers (●, ○, Δ) print twice and a header row sits inside the table body — the price of
  two self-contained blocks.
- A probability column shares its width with an unrelated race column; the alignment is what
  makes the pair read as one object.
- The Δ row is present in a mounted block before values land (ADR-0017 rule 3).
- Rejected: compacting padding and merging columns (still over budget in de, it, ru, es).

## Guard

`frontend/tests/e2e/eval-panel-no-scroll.spec.js` (facts and decision share a row at default
size; facts are two bodies of one table).
