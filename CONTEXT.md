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

**Flagged Position**:
A Position the user marked as worth studying *in the tool the match came from* — today
only eXtreme Gammon, which records it per move. Like the individually-imported property it
is sticky, never part of the Zobrist hash, and never set or cleared by a gesture inside
blunderDB: it is a fact of the source file. So a Position flagged in one Match stays flagged
even when a later import brings the same Position in unflagged.
A flagged cube decision marks *both* Positions blunderDB derives from it — the double and
the take/pass — because the source records one decision and blunderDB splits it in two.
_Avoid_: bookmark, starred, favourite — a Flagged Position is durable and read-only; a
transient "come back to this" list is a Collection.

**Orphan purge**:
The sweep that runs when a Match is deleted: each Position the Match referenced is removed
unless something else still holds it. What "holds" a Position is a deliberate list — another
Match's move, Collection membership, an Anki card, or being individually imported. Neither an
Analysis nor a Comment holds a Position: both can arrive *with* the Match (importers attach the
source file's per-move notes as Comments), so neither is evidence the user did anything. A note
the user wrote on a Match-sourced Position is therefore still lost when the Match is deleted —
to keep such a Position, put it in a Collection or save it.

### Knowing what a position is worth

**Analysis**:
A *record* of what an engine concluded about a Position, stored against it and read back
later. It carries its provenance (`AnalysisEngine`, e.g. XG, GNUbg, BGBlitz, gammonNet) and
its `AnalysisDepth`. A Position has at most one Analysis row; several engines coexist inside
it, each entry tagged with the engine that produced it. An Analysis is *relit*, never
recomputed: a Position with no identity — one the user has just composed on the board — cannot
have one.
_Avoid_: evaluation, eval, engine output

**Evaluation**:
The *result of a computation* the embedded engine performs on the position currently on the
board, whatever it is, saved or not. It has no identity, is never loaded from the database,
and is valid only for as long as the position does not move. An Evaluation may later be
written down and so *become* an Analysis; until it is, it is not one.
_Avoid_: analysis, live analysis, instant analysis

The distinction is the reason two panels exist: one reads records, the other computes. They
render similar tables and share the components that draw them, but they are not two views of
one thing.

**Network**:
The weights, and only the weights — `strehl-prob5-512-512-256-128`. A network changes name
only when its weights change: neither the search wrapped around it, nor a quantisation, nor a
port to another language makes it a new one.

**Configuration**:
A network *plus* the search, the endgame tables and the match-equity table around it — the
whole of what produces a number. `gammonNet 2-ply` names a Configuration, not a Network. Two
Configurations sharing a Network are still two Configurations.

**Canonical parameters**:
The Configuration blunderDB writes down: 2-ply, pruning `k=12`, Kazaross-XG2. What the user
adjusts for comfort while reading the board is a different setting, and it never reaches the
database.

**Regime**:
Which kind of answer the panel is giving about a race, always stated on screen, never inferred
by the reader:
- *exact* — read from a two-sided bearoff database. A lookup. The answer.
- *evaluated* — computed by the engine, which plays the trajectory out. Not a lookup, not a
  guess either.
- *estimated* — a snapshot summary of a trajectory. Legitimate for a win probability
  (convolution plus a calibrated correction), and **never** used for a cube verdict.

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

A Comment carries no provenance: text the user typed and text an importer lifted from a
source file are indistinguishable. So "the Position is commented" means only *some* non-empty
text is attached — never "the user annotated it". Empty text is not a Comment: a row whose
text is `''` counts as absent everywhere (search, listing, export).

Match and Tournament each carry their own comment field. Those are annotations of the Match
or the Tournament, not of its Positions: a commented Match does not make its 300 Positions
commented, and no Position-level rule in this glossary reads them.

### Players

**Player**:
A name exactly as it appears in an imported Match (`player1_name`/`player2_name`).
blunderDB has no notion of a person behind the name: the same human signing under
two spellings is two Players, and every per-player statistic, list or table is keyed
by this literal name. Merging spellings is a destructive user gesture (merge players),
never an inference. Distinct from Tenant — the Tenant owns the data; a Player merely
appears in it.
_Avoid_: user, account, identity

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

### Handing a database to someone else

**Watermark**:
The producer's signed statement of where a database comes from — what it is, who made it,
and whatever they chose to attach to it (terms of use, a contact address). It is applied at
export and nowhere else, and it is the mirror image of the database's own metadata: `user`
and `description` belong to the Holder and are theirs to edit, a Watermark belongs to the
producer and is read-only everywhere else.
It is **tamper-evident and unforgeable, never unremovable** — a Holder with SQL tools can
delete it, and the design says so out loud rather than pretending otherwise.
_Avoid_: licence, DRM, protection — a Watermark forbids nothing and identifies nobody; it
attributes a file to its source.

**Issuer identity**:
The durable identity a Watermark is signed with. It belongs to a person, not to a database:
every file the same producer marks carries the same public fingerprint, which is what lets a
recipient check that a file really came from them. It comes into existence without being
asked for, on the first watermarked export, and moves from one machine to another as a
single file.

**Protected file**:
An exported database wrapped in an encrypted container, opened once with a password and
thereafter an ordinary database. It protects the file **in transit** — the stray copy, the
attachment forwarded by mistake — not the database, since whoever the password was given to
can open it. Its header is cleartext, so a Watermark stays readable without the password.

**What is deliberately absent**:
Nothing records who holds a database, who opened it, or where its contents came from. The
recipient's side writes nothing at all. This is a decision, not an omission: see ADR-0007.

## The host environment

**Host capability**:
A facility blunderDB consumes from the machine, OS or desktop it runs on but does **not**
own — its presence and its shape are not guaranteed and vary from system to system.
Examples: an image clipboard tool, an installed font, the keyboard layout, the locale, a
writable config/data directory, the WebView renderer. Each host capability has a *state*
(present / absent / degraded) and a *fallback strategy*.
_Avoid_: dependency (reserved for Go/npm packages), platform (too coarse — clipboard tools
and keyboard layouts vary *within* one platform).

**Essential capability**:
A Host capability without which blunderDB cannot fulfil its core mission of storing and
searching positions. There are exactly two: a **writable config/data directory** and the
**WebView renderer**. When one is absent, blunderDB fails loud and early with an actionable
message rather than entering a half-broken state.

**Optional capability**:
A Host capability whose absence must never block the core product — an image clipboard,
a CJK font, single-instance locking, an expected keyboard layout, a specific locale. When
absent or in a different shape, blunderDB degrades gracefully: it detects, falls back on an
embedded or native substitute where possible, and surfaces a non-blocking notice rather than
an interrupting error.

**Fallback strategy**:
The ordered ladder (its rungs) blunderDB walks when an Optional capability is absent or
degraded: prefer a substitute it *ships* (an embedded font) or a *native* mechanism (the
WebView's own clipboard) over requiring the user to install an external tool, and only then —
if every rung fails — show a non-blocking notice explaining what is unavailable and how to
restore it.

**Capability probe**:
The thin piece of code that inspects the host and reports the raw *facts* about one Host
capability's state as plain data — e.g. "xclip present, wl-copy absent, session is Wayland".
A probe does no deciding: it only gathers facts (a `LookPath`, a `Getenv`, a `Stat`) and
carries no fallback logic of its own. Kept deliberately dumb so it needs only a smoke test;
the decision it feeds lives in the Fallback policy.
_Avoid_: detector (too vague — a probe reports, it does not choose).

**Fallback policy**:
The pure function that turns a Capability probe's facts into a chosen rung of the Fallback
strategy — facts in, decision out, no I/O. Because all the *risk* (which rung is right) lives
here and it touches nothing external, it is exhaustively unit-testable with hand-written fact
values, without simulating a whole host. Pairs with a Capability probe per capability
(clipboard, font, locale, paths).
