# blunderDB

blunderDB stores backgammon positions and the engine analyses attached to them, so a
player can search their own blunders across the matches and positions they have imported.

## Language

### Positions and their origin

**Position**:
A backgammon decision point: board, cube, dice, score and match flags. Identified by its
Zobrist hash, so the same position imported twice is one row, never two.
_Avoid_: board, node, entry

**Deduplication**:
The rule that a Position's identity is its Zobrist hash. Any import that produces an
already-known Position lands on the existing row and enriches it (analyses merged,
comments appended) rather than creating a second one.

**Individually imported Position**:
A Position that entered the database on its own — written from the board, or read from a
position file — as opposed to arriving as part of a Match. Because of Deduplication, this
is a *sticky* property: a Position that was individually imported at least once keeps the
property forever, even if a Match containing it is imported afterwards. It is set by the
import that created or re-touched the row, and is never set or cleared by a user gesture.
_Avoid_: manual position, hand-added position, favourite, marked position

**Match-sourced Position**:
A Position reachable from a Match through the `move` → `game` → `match` chain. Not the
complement of "individually imported": a Position can be both.

**Orphan purge**:
The sweep that runs when a Match is deleted: each Position the Match referenced is removed
unless something else still holds it. What "holds" a Position is a deliberate list — another
Match's move, Collection membership, an Anki card, or being individually imported. Neither an
Analysis nor a Comment holds a Position: both can arrive *with* the Match (importers attach the
source file's per-move notes as Comments), so neither is evidence the user did anything. A note
the user wrote on a Match-sourced Position is therefore still lost when the Match is deleted —
to keep such a Position, put it in a Collection or save it.

### Sets of positions the user curates

**Collection**:
A named, ordered set of Positions the user assembles by hand. Membership is a user
gesture, unlike the individually-imported property.

**Anki deck**:
A set of Positions turned into spaced-repetition cards.

**Tag**:
A `#word` inside a Position's Comment. There is no tag table — tags are a convention
inside comment text, searchable only as substrings.

**Comment**:
Free text attached to a Position. The model allows several per Position (match import adds
one row per note found in the source file), but the GUI treats a Position as having a
single comment: it loads and edits whichever row comes back first. Known debt — a Position
that arrived with two comments shows only one of them, arbitrarily.

### Who owns the data

**Tenant**:
The owner of a set of Positions, Matches, Collections and decks. On the desktop
there is exactly one, implicit Tenant: the person whose database file it is. In
server mode each caller is a distinct Tenant, and nothing one Tenant stores is
ever visible to another. Deduplication, the Orphan purge, and every other rule
in this glossary apply *within* one Tenant — the same board position stored by
two Tenants is two rows, not one.
_Avoid_: user, account, customer

**Scope**:
The storage layer's spelling of Tenant: every persistence call carries a scope,
and the empty scope denotes the desktop's single implicit Tenant. "Scope" and
"Tenant" name the same concept; prefer Tenant in prose and design discussion.
