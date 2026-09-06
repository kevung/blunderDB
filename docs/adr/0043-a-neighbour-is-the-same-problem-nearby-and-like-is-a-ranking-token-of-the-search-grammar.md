# A neighbour is the same problem nearby, and `like` is a ranking token of the search grammar

## Status

accepted — decided 2026-09-07 in a design interview, after measuring what the `like`
command shipped by J.3 (#293, commit 2399c479d) actually returns. Revises that commit's
three choices (raw distance, a command apart, no combination with tokens) and keeps its
metric and its exhaustive scan. Glossary: *Neighbouring Position* in `CONTEXT.md`.

## Context

`like` was shipped as a command that ranks the whole library by the 1-D transport
distance of the checker vectors, seen from the side on roll, with no other criterion,
and refused to combine with the search tokens (« the closest *among* those that match
would be another question »). The metric is sound — the number reads as checker-pips —
but measured on the demo library (757 positions, 3 matches) the ten neighbours of any
position were:

| target | neighbours (distance) |
|---|---|
| game 1 ply 39, cube | ply 40 (0), 36 (9), 35 (9), 43 (10), 44 (10), 48 (29)… |
| game 4 ply 5, cube | ply 6 (0), 2 (15), 1 (15), then other games at 15–17 |
| game 7 ply 19, checker | plies 21, 23, 17, 15 (11–21), then game 1 at 27 |

Three facts, none of them a bug of the metric:

- **the first neighbour is always at distance 0**: a cube decision and the checker play
  that follows it are two rows on the same board, and `like` called them the same
  position;
- **the next ones are the plies before and after, in the same game**: two plies are one
  roll, 8 to 16 checker-pips, and no other game gets that close. On a library of imported
  matches — everybody's library — `like` answered « here is the game you are looking at »;
- **nothing of the decision enters the distance**: not the kind of decision, not the
  regime (money or match), not the dice.

The J.3 sheet had asked for a prototype with a player judging « close by the metric =
close by eye » before choosing; the commit leaned on report P7 alone and that check never
happened. The interview replaced it: the question the user asks is « have I met **this
problem** elsewhere? », not « which drawing is nearest? ».

## Decision

1. **A neighbour is the same problem with a nearby structure.** The transport distance
   stays, unchanged and alone in the number shown. What changes is the **set it ranks**:
   the target's *equivalence class* — same kind of decision (checker or cube); for a cube
   decision, the same regime (money or match); and, when the target belongs to a match,
   **another match**. Dice, score and cube value stay outside both distance and class:
   the ordinary tokens (`D`, `cube`, `score`, `s`…) narrow on them when wanted. No
   penalty is ever folded into the distance — one unit, readable, is the only thing the
   metric has going for it.

2. **`like` is a token of the search grammar, and the command goes.** `s like`, `s like42`,
   with `like<n>` as a distance ceiling and `*` widening the class:

   | form | meaning |
   |---|---|
   | `like` | neighbours of the current position, in its class |
   | `like42` | neighbours of position 42 |
   | `like<12` | none beyond 12 checker-pips (overrides the preference) |
   | `like42*` | class widened: every kind of decision, both regimes |

   It is the first token that **ranks** instead of filtering, and the grammar says so in
   one sentence: the presence of `like` orders the result by ascending distance; every
   other token narrows the set it ranks. Everything that reads a query understands it at
   once — `ss like` is « the closest *among the displayed list* », a live collection can
   be « the neighbours of 42 with `E>80` », the search history replays it, the CLI takes
   it through `--query`, and `--like` disappears. A command that accepted tokens would
   have been a second parser and a second path to the search.

3. **A drawn board is a valid target.** In EDIT mode `s like` ranks the stored positions
   against what is drawn; the class is read on the drawing (dice set → checker decision,
   none → cube; money or match from the score) and « another match » does not apply. This
   is the question the exact structure search believed it was asking — « I vaguely
   remember a position like this » — and it costs nothing: `Similar` already takes a
   `Position`, not an id.

4. **A ranking is bounded, and an empty ranking is empty.** At most *k* neighbours, never
   beyond the asked distance, and when nothing passes the ceiling the list is empty and
   says so — never ten unrelated positions with a figure in the status bar. *k* (raised
   from ten to a few dozen: a list is browsed, not read at a glance) and the default
   ceiling are **preferences** of the configuration panel, in the persisted `Config`;
   `like<n>` overrides the ceiling for one query. No built-in ceiling: the scale depends
   on the phase (ten pips is nothing in a race and another position in the opening) and
   nobody has measured it.

5. **The distance is shown on each neighbour, where the position is read.** The
   explanation line under the analysis tables (#298) says « at 9 checker-pips from
   position 40 » for every neighbour; the status bar's range survives no gesture. Entry
   points: the token, a « Neighbouring positions » item in the board's context menu, and a
   keyboard shortcut — the gesture is « I am looking at a position and wonder whether I
   have seen it before ». No dedicated panel: it would be a second way of browsing a set,
   which the service rightly refused.

## Considered options

- **Keep the raw distance and let the user read the ids** — rejected: on a library of
  matches the result is the game being looked at, every time.
- **Exclude a window of plies around the target instead of the whole match** — rejected:
  one more arbitrary setting, for the rare case of a position met twice in one match
  twenty moves apart.
- **Same kind of decision imposed, not widenable** — rejected in favour of a default with
  `*`: « every kind » is a legitimate, rarer question, and a glyph costs one line of
  documentation in nine languages, not a word.
- **Regime as a penalty in the distance** — rejected: two units in one number, and an
  arbitrary exchange rate.
- **A built-in distance ceiling** — rejected until measured per phase.
- **`like 42 E>80` as a command taking tokens**, or both spellings — rejected: a second
  parser, and two spellings of one thing.
- **A panel of thumbnails** — rejected: a second way of browsing a set, and a page of
  manual in nine languages.

## Consequences

- `engine/similarity.go` is unchanged; `sqlshared.Similar` gains the class (decision
  kind, regime, excluded match id) and the ceiling; the contract test in `storagetest`
  covers the class and the empty ranking.
- `searchquery` parses `like[id][<n][*]`; `Parse` marks the query as ranked; both
  backends order by distance when it is; `ss` and live collections get it for free.
- `similarService.js`, the `like` command, `--like` and `/v1/positions.similar` go; the
  route's body becomes a query like any other.
- Two `Config` fields (`like_limit`, `like_max_distance`), a configuration entry, a
  context-menu item, a shortcut, an explanation-line message; `cmd_mode.rst`,
  `manuel.rst`, `raccourcis.rst` and their eight catalogues; `make help`.
- The J.3 sheet's unmet check is due now, not later: a short corpus (a few dozen
  targets from a real library) where the interview's author judges whether the first
  neighbours are « the same problem » — before the token is documented, because the
  class rules are the thing that check validates.
