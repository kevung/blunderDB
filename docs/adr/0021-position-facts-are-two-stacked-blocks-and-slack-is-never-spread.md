# Position facts are two stacked blocks on one grid, and slack is never spread between blocks

## Status

accepted — decided 2026-09-01, the day after ADR-0018/0019/0020. **Extends** ADR-0017 rule 1 and
ADR-0020 rule 8; reverses nothing, moves no number, changes no scale, touches no engine and no
schema. It is a layout decision and only a layout decision.

ADR-0018 closed on a warning about repeated axis decisions on the same panel, and ADR-0020 answered
it rule by rule. The same test applies here:

- **ADR-0017 rule 1** (facts on one side, the decision on the other) — untouched, and the reason
  this decision is possible at all: the facts were already separated from the decision, they were
  simply all rendered in one table.
- **ADR-0017 rule 2** (the board asks exactly one question) — untouched, and leaned on below: it is
  what makes the vertical space free.
- **ADR-0018 rule 1** (`PositionFactsTable` carries per-side facts only) — untouched.
- **ADR-0018 rule 2** (the pre-roll vector takes the axis of the list it heads) — untouched. When
  there is a list, the probability block is not rendered here at all; that is unchanged.
- **ADR-0018 rule 6 / ADR-0020 rule 7** (Défi has three zones) — unchanged in effect: a side is one
  zone across both blocks, so revealing the bottom side reveals it in the probabilities and in the
  race numbers at once, which is what a "side" has always meant.
- **ADR-0020 rule 8** (the strip is a full-width line, so nothing is pushed to a far edge) —
  *generalised* here from the badge strip to every block the panels lay out.

What is new is not an axis. It is that **the two kinds of position fact were welded into a single
table wider than the panel that has to hold it**, and that one panel still spread its slack between
blocks instead of leaving it at the end.

## Context

### The cube block falls under the facts in seven languages out of nine

At blunderDB's default window (1024 px, `config.go:298`) the Eval panel has 996 px of content
width. `.top-row` is a `flex-wrap` row of two children, facts then decision, so the decision — last
— is the only child that can wrap. Measured in the running app (Playwright, viewport 1024, race
position with a cube question, longest verdict), before and after this decision:

| language | facts (10 columns) | cube | total | before | stacked facts | total | margin |
| --- | --- | --- | --- | --- | --- | --- | --- |
| en | 740 | 208 | 968 | same line | 446 | 674 | 322 |
| fr | 709 | 234 | 963 | same line | 434 | 688 | 308 |
| ja | 752 | 232 | **1004** | wrapped | 451 | 703 | 293 |
| it | 782 | 255 | **1057** | wrapped | 489 | 764 | 232 |
| fi | 804 | 237 | **1062** | wrapped | 532 | 789 | 207 |
| de | 815 | 255 | **1089** | wrapped | 478 | 752 | 244 |
| ru | 820 | 253 | **1093** | wrapped | 527 | 799 | 197 |
| es | 880 | 221 | **1121** | wrapped | 561 | 802 | 194 |
| el | 829 | 276 | **1125** | wrapped | 528 | 824 | 172 |

Seven languages out of nine wrapped unconditionally. The two that did not had ~30 px of margin,
which the *content* then spends: a cube already turned once relabels the three options
«Redouble…» instead of «Double…» and costs 25 px in French, and a three-digit pip count costs
another few. So even in the source language the layout was decided by the position on the board —
same panel, same window, cube block sometimes beside the numbers and sometimes under them.

Under it, the panel was spending height it does not have to save width it does: the content row
went from 108 px to 201 px when it wrapped, in a panel 250 px tall by default (`config.go:119`),
while 250 px of width sat unused to the right of the facts.

Compacting was measured before being rejected: cell padding 10 → 7 px plus merging «Jets moy.» and
«Écart-type» into one `7.475 ± 1.943` column brings French comfortably under budget but leaves de,
it, ru and es still over it. Any fix that has to be re-measured per language is not a fix; it is a
deferral to whoever next writes a translation.

### The Analysis panel spread its slack down the middle

`AnalysisPanel.svelte`'s `.tables-container` carried `justify-content: space-between`. With three
children (facts, decision, depth/engine footer) and ~300 px of slack, that slack was inserted
*between* the blocks: the facts hugged the left edge, the footer the right, and the panel showed a
band of white through its middle. This is the exact failure ADR-0020 rule 8 removed from the Eval
panel's badge strip (`margin-left: auto`), one component over, still in place.

## Decision

1. **`PositionFactsTable` renders two stacked blocks, not one line of ten columns.** The
   probability block (win, gammon, backgammon, cubeless) and the race block (EPC, pip count,
   wastage, mean rolls, standard deviation) are two `<tbody>`s, each with its own header row and
   each complete on its own: the same three rows — bottom, top, Δ — under its own headers.

2. **The two blocks share one column grid.** They are bodies of ONE table, not two tables stacked:
   two tables size their columns independently and come out different widths, with nothing lining
   up under anything — precisely the "disparate sizes" a stacked layout invites. Sharing a grid
   gives them the same left edge, the same right edge, the same column stops and a single column of
   side markers; the grid is as wide as the wider block (five value columns), so the probability
   block ends in one empty cell, and that empty cell is what buys the alignment. The vertical rule
   that used to divide probabilities from race columns is gone: the blocks are separated by a line
   of air, not by a border.

3. **Height is spent here because it is free here.** A cube decision is shown only when there are
   no dice (ADR-0017 rule 2), and the candidate list exists only when there are — so the two never
   compete. The content row becomes 178 px in all nine languages: more than the 108 px of the rare
   unwrapped case, less than the 201 px of the wrapped one, and inside the panel's budget with no
   overflow anywhere (measured, all nine). When there are dice, `showProbabilities` is false, only
   the race block is rendered, and the list keeps the height it had.

4. **Blocks are laid out at a constant gap; leftover width is left over.** No panel distributes its
   slack between blocks — no `space-between`, no `margin-left: auto` on a content block. Slack
   accumulates at the end of the row, where it reads as margin. This restates ADR-0020 rule 8 for
   every block rather than for the badge strip alone, and it is why `.tables-container` in the
   Analysis panel now lays its three tables out left to right at the same 20 px gap the Eval panel
   uses.

5. **Wrapping stays, as a fallback, and is no longer the normal case.** `.top-row` keeps
   `flex-wrap`: a panel narrowed by hand below the stack-plus-decision width still stacks rather
   than clipping. What changes is that the default window no longer triggers it in any language —
   the widest case (el) needs 824 px of the 996 available, leaving a 172 px margin.

## Consequences

- The Eval panel's cube decision sits beside the facts at the default window size in all nine
  languages, with 172–322 px of margin. A translator has real headroom to work in, and no longer
  changes the layout by writing a longer word; neither does a pip count reaching three digits.
- Two blocks mean the side markers (●, ○, Δ) are printed twice, and a header row appears inside the
  table body. That is the price of the stack, and it is what makes each block readable on its own;
  the shared grid and the fixed-width label column keep the repetition reading as one axis rather
  than as two tables.
- Sharing a grid means a probability column is as wide as the race column above or below it — «Gain
  (%)» sized against «EPC», and so on. They are unrelated quantities sharing a stop, which is a
  deliberate trade: the alignment is what makes the pair look like one object.
- The Δ row is now always present in a mounted block, even before values land, instead of appearing
  when the first value arrives. This is ADR-0017 rule 3 applied more strictly than before — the
  block no longer changes height under the reader.
- The Analysis panel's middle band is gone: its tables now follow one another at a 20 px gap and
  the leftover width sits at the right. The change is visible as blocks moving left, not as
  anything new appearing.
- The e2e no-scroll test (`eval-panel-no-scroll.spec.js`, owed by ADR-0017, paid by ADR-0020) gains
  the assertion this decision needs: at the default panel size, in a race cube position, the facts
  stack and the decision block share a row, and the facts are two bodies of one table. Measured,
  because a layout property nobody measures after four layout changes is not a property.
