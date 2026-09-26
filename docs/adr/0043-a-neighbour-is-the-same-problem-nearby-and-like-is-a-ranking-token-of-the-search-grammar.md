# A neighbour is the same problem nearby, and `like` is a ranking token of the search grammar

Status: accepted.

## Context

Ranking the whole library by the 1-D transport distance of the checker vectors (seen from the
side on roll) answers the wrong question. On a library of imported matches the first neighbour
is always at distance 0 (the cube decision and the checker play that follows share a board),
and the next ones are the plies before and after in the same game — one roll is 8 to 16
checker-pips, and no other game gets that close. The user asks « have I met this problem
elsewhere? », not « which drawing is nearest? ». Glossary: *Neighbouring Position* in `CONTEXT.md`.

## Decision

1. **A neighbour is the same problem with a nearby structure.** The transport distance
   (`engine/similarity.go`) is unchanged and alone in the number shown. It ranks only the
   target's equivalence class: same kind of decision (checker or cube); for a cube decision,
   the same regime (money or match); and, when the target belongs to a match, another match.
   Dice, score and cube value are outside distance and class — ordinary tokens narrow on them.
   No penalty is ever folded into the distance.
2. **`like` is a token of the search grammar**, not a command:

   | form | meaning |
   |---|---|
   | `like` | neighbours of the current position, in its class |
   | `like42` | neighbours of position 42 |
   | `like<12` | none beyond 12 checker-pips (overrides the preference) |
   | `like42*` | class widened: every kind of decision, both regimes |

   It is the one token that ranks instead of filtering: its presence orders the result by
   ascending distance, every other token narrows the set it ranks. Everything that reads a
   query understands it — `ss like`, live collections, search history, CLI `--query`.
3. **A drawn board is a valid target.** In EDIT mode `s like` ranks stored positions against
   the drawing; the class is read on it (dice set → checker, none → cube; regime from the
   score) and "another match" does not apply.
4. **A ranking is bounded, and an empty ranking is empty.** At most *k* neighbours, never beyond
   the ceiling; when nothing passes, the list is empty and says so. *k* and the default ceiling
   are preferences in the persisted `Config` (`like_limit`, `like_max_distance`); `like<n>`
   overrides the ceiling for one query. No built-in ceiling: the scale depends on the phase and
   is unmeasured.
5. **The distance is shown on each neighbour, where it is read**: the explanation line under
   the analysis tables says « at 9 checker-pips from position 40 ». Entry points: the token, a
   « Neighbouring positions » item in the board's context menu, and a keyboard shortcut. No
   dedicated panel.

## Consequences

- `sqlshared.Similar` takes the class (decision kind, regime, excluded match) and the ceiling;
  `searchquery` parses `like[id][<n][*]` and marks the query as ranked; both backends order by
  distance.
- There is no `like` command, no `--like` flag and no `/v1/positions.similar` route.
- The class rules are validated against a human-judged corpus (`cmd/likecorpus`).
- Rejected: raw distance over the whole library — returns the game being looked at.
- Rejected: excluding a window of plies instead of the whole match — one more arbitrary setting.
- Rejected: same kind of decision imposed, not widenable — `*` costs a glyph, not a word.
- Rejected: regime as a penalty in the distance — two units in one number.
- Rejected: a built-in distance ceiling — unmeasured per phase.
- Rejected: a `like` command taking tokens, or both spellings — a second parser.
- Rejected: a panel of thumbnails — a second way of browsing a set.

## Guard

`pkg/blunderdb/storage/storagetest/contract_similar.go`,
`pkg/blunderdb/searchquery/corpus_test.go`, `cmd/likecorpus/`.
