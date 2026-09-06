# A transcription is a draft that owns its match

## Status

accepted — decided 2026-09-07, with 0043. Leans on 0036 (the trash snapshots gestures,
and a replacement is not one), 0013 (evaluations fill gaps), 0001 and
0028 (a Position's identity is its Zobrist hash, and what is not in the hash).

## Context

A match being transcribed changes at every keystroke, and it changes *backwards*: a die
misread three turns ago is corrected, and every position after it moves. The database,
on the other hand, is built for matches that arrive whole and never change: positions
are deduplicated by hash across every match, a match is hashed by its content, deleting
a match purges the positions nothing else holds, and a growing share of deletions is
snapshotted into the trash for thirty days.

Writing a transcription straight into `match`/`game`/`move` would therefore make every
correction a delete-and-reinsert through the dedup and the orphan purge; the
match hashes would be meaningless until the match is finished; and a half-typed match
would count in the Performance Rating and appear in every search.

The transcription must also survive a crash of the application: a user forty moves into
a match will not start again.

## Decision

1. **A Transcription is a draft, distinct from the Match it produces.** It is one row of
   a dedicated table, holding a JSON document that carries its own format version, so a
   change to the document's shape is a version of the document and never a
   `DatabaseVersion` migration. The row is written after every Action the user records,
   in its own transaction: an application crash loses nothing that was committed. The
   undo stack stays in memory.

2. **Saving materialises a Match; saving again replaces it, and the draft stays the
   source of truth until the user closes it.** The replacement is atomic, keeps the
   Match's `id` — a tournament, a living collection, the last-visited position all point
   at it — and goes through `ingest.WriteMatch`, the one path that creates matches.
   Positions of unchanged Actions land on their existing rows by deduplication, so a
   Comment, an Anki card or a Collection membership attached during the review survives
   the correction of a move elsewhere in the match. Positions of corrected Actions are
   purged by the ordinary retention rule.

3. **The replacement does not pass through the trash, now or later.** Today deleting a
   match writes no snapshot at all — `DeleteMatch` cascades and runs the orphan purge,
   and 0036 snapshots positions, collections and comments only. The replacement follows
   that same path. The day a match deletion is snapshotted, the replacement stays out of
   it: replacing one match twenty times during a review is not twenty deletions, and the
   trash would otherwise hold twenty copies of one match — the same exemption 0036 grants
   the orphan purge, housekeeping rather than a gesture. Closing a draft that was never
   saved deletes the draft row, with a confirmation and no snapshot either: there is
   nothing saved to put back.

4. **An Action is one player's act**, never a pair: a double is one Action, the take or
   pass that answers it is another, each with its own Position and Decision, as XG
   records them and as the `.mat` lays them out. The side of an Action is *proposed* by
   the engine at entry (the trait alternates; the opening's two dice say who starts) and
   *owned* by the Action once recorded: inserting or deleting an Action therefore never
   flips the side of everything after it. It creates one local inconsistency instead.

5. **Every inconsistency is derived, kept and marked — never stored as a flag, never
   deleted.** An illegal move is a board `LegalMoves` cannot reach from the previous
   position; a double turn is two consecutive Actions of one side; an impossible cube
   action is a double by a player who does not hold the cube, or an answer with no
   offer; an Action past the end of the match appears when the match length is
   shortened. Each is recomputed by replaying the draft from the corrected Action on,
   which is what a correction triggers. The saved Match carries none of them as data: an
   illegal move is a Move whose notation is what was played and whose Position is the
   board it left, and the fact is derivable by anyone with the previous position and
   `LegalMoves` — the same rule 0035 applies to the game phase.

6. **A resignation is an Action of the draft and a fact of the Game, not a Move.** The
   saved Game carries `winner` and `points_won`; no row is added to `move`, no column to
   `game`. The `.mat` format has no resignation token and re-imports only the result;
   whether a game ended by a bear-off or by a resignation is recoverable from the last
   board. A match the user abandons is an unfinished Match, `winner = -1` on its last game.

7. **Crawford is derived from the sequence of scores and written as the glossary's
   sentinel** — away `1` for the Crawford game, `0` for the games after it. The importers
   never write that sentinel (every 1-away position reads as Crawford, cube dead), which
   is a defect of theirs to be fixed with a rehashing repair, not a convention for a new
   writer to inherit: the first consumer of a transcribed position is the cube analysis,
   and post-Crawford the trailer's opening double is the position that matters.

8. **Saving starts the canonical analysis, scoped to the match.** The 2-ply batch runs on
   the saved match's positions that have no analysis (0013 fills gaps, so a re-save after
   a correction analyses only what changed). The candidates shown while typing are a
   0-ply Evaluation and are never written. Closing the application cancels a running
   batch before the database is closed — a fix to `shutdown` that every batch benefits
   from. No "analysis pending" state is stored: on reopening the library, a transcribed
   match with unanalysed positions is a fact the panel derives, announces, and offers to
   finish.

9. **The engine of the transcription lives in Go**, in a pure package
   `pkg/blunderdb/transcript/` with no SQL, in the manner of `issuance`: a document, the
   gestures that change it, and an annotated document in return (positions, sides,
   inconsistencies, scores, Crawford, game ends). Its one internal dependency is
   `domain` — `ingest` imports `storage`, so the package does not import `ingest`: it
   returns the match, games and moves of the domain, which is what a save turns into a
   graph and what the `.mat` renderer already takes. The Svelte panel is a
   client: each gesture returns the whole annotated document and the store displays it.
   The CLI gains `transcribe` over the same package; `serve` exposes nothing (0039).

## Considered options

- **Write the match into `match`/`game`/`move` as it is typed.** Rejected for the
  reasons of the Context: every correction becomes a cascade through dedup and purge;
  the match hashes are unstable; a half-typed match enters the statistics.
- **A draft in a file next to the configuration (XDG).** Rejected: the draft names two
  Players, a tournament and a length of *this* library, and opening another library
  would show the wrong draft; the Match panel's "draft in progress" line needs a query,
  not a file; a copied or backed-up library would leave its drafts behind.
- **A final save that closes the draft.** Rejected: nothing in blunderDB edits the moves
  of an existing Match, so a correction discovered after the save — the common case in
  a review — would have no path at all.
- **A `Double/Take` pair as one Action**, as the gnubg importer stores it. Rejected: the
  cursor must be able to stand between the double and its answer, and the two are two
  Positions with two Decisions; the importer's combined form is a debt, not a model.
- **Derive the side from the previous Action at every replay.** Rejected: deleting a
  move typed twice would flip the side of every later Action and make each of them
  illegal, turning a one-cell correction into a rewrite of the match.
- **An `illegal` column on `move`.** Rejected under rule 5: true at write time, false the
  day an importer or a repair forgets it; derivable at zero cost from the previous
  position.
- **The transcription engine in JavaScript**, calling Go only for legal moves. Rejected:
  the rules exist once, tested and held identical to gammonNet's generator by a
  differential test; a second copy is the debt the import unification closed, and a
  Wails round trip costs a millisecond against a keystroke.

## Consequences

- `DatabaseVersion` 2.21.0: one table, migrated in the three copies (`database`,
  `storage/sqlite`, `storage/postgres`) although the server never uses it, and
  `demo.db` regenerated. The table is on the allow-list of a full database export (the
  drafts are the user's work) and off the list of a filtered export.
- `ingest.WriteMatch` gains a replace mode that keeps a match id and runs the ordinary
  orphan purge; the retention predicate is untouched.
- `RenderMAT` writes the headers the parser already reads (Site, Round, EventDate,
  Transcriber); a `.mat` containing an illegal move is exported with a warning that
  gnubg and XG will flag the move and diverge from there, never refused.
- A twin issue is opened for the importers to write the Crawford sentinel, with a repair
  that rehashes the affected positions; until then a transcribed post-Crawford position
  does not deduplicate with its imported twin, which is a loss of nothing (the twin was
  wrong).
- `shutdown` cancels a running gammonNet batch before closing the database.
- The glossary gains Transcription, Action, Cursor, Replay and Inconsistency, and says
  what a Transcription is not.
