# A transcription is a draft that owns its match

Status: accepted.
See also: ADR-0044, ADR-0036, ADR-0013, ADR-0001, ADR-0028, ADR-0035.

## Context

A transcribed match changes at every keystroke, and backwards: a die misread three turns ago
moves every later position. The database is built for matches that arrive whole — positions
deduplicated by hash, matches hashed by content, orphan purge on delete. Writing a
transcription straight into `match`/`game`/`move` would make each correction a cascade, the
hashes meaningless, and a half-typed match part of every statistic. The draft must survive a
crash.

## Decision

1. **A Transcription is a draft, distinct from the Match it produces**: one row of a
   dedicated table holding a JSON document with its own format version (a shape change is a
   document version, never a `DatabaseVersion` migration). The row is written after every
   recorded Action, in its own transaction. The undo stack stays in memory.
2. **Saving materialises a Match; saving again replaces it; the draft stays the source of
   truth until closed.** The replacement is atomic, keeps the Match `id`, and goes through
   `ingest.WriteMatch`. Positions of unchanged Actions land on their existing rows (comments,
   Anki cards, collection memberships survive); positions of corrected Actions go by the
   ordinary retention rule.
3. **The replacement never passes through the trash**, even the day match deletion is
   snapshotted — replacing twenty times is housekeeping, as ADR-0036 exempts the orphan
   purge. Closing a never-saved draft deletes its row after confirmation, with no snapshot.
4. **An Action is one player's act**: a double and its answer are two Actions, each with its
   Position and Decision. The side is proposed by the engine at entry and owned by the Action
   once recorded, so inserting or deleting never flips later sides; it creates one local
   inconsistency instead.
5. **Every inconsistency is derived, kept and marked — never stored, never deleted**: illegal
   move, double turn, impossible cube action, Action past the end of the match. Each is
   recomputed by replaying from the corrected Action. The saved Match carries none as data; an
   illegal move is a Move whose Position is the board it left.
6. **A resignation is an Action of the draft and a fact of the Game** (`winner`,
   `points_won`), not a Move. An abandoned match is unfinished: `winner = -1` on its last game.
7. **Crawford is derived from the score sequence and written as the glossary's sentinel**:
   away `1` for the Crawford game, `0` after it. Every writer uses it; `blunderdb repair`
   rehashes positions written without it.
8. **Saving starts the canonical 2-ply analysis, scoped to the match's unanalysed positions**
   (ADR-0013). Candidates while typing are a 0-ply Evaluation, never written. Shutdown cancels
   a running batch before closing the database. No "analysis pending" state is stored: the
   panel derives it and offers to finish.
9. **The engine lives in Go**, in the pure package `pkg/blunderdb/transcript/` (no SQL, depends
   only on `domain`, not on `ingest`): a document, its gestures, and an annotated document in
   return. The Svelte panel is a client; the CLI has `transcribe`; `serve` exposes nothing.

## Consequences

- One table, migrated in `database`, `storage/sqlite` and `storage/postgres`; on the
  allow-list of a full export, off a filtered one.
- `ingest.WriteMatch` has a replace mode that keeps the id; the retention predicate is
  untouched.
- `RenderMAT` writes Site, Round, EventDate, Transcriber; a `.mat` with an illegal move is
  exported with a warning, never refused.
- Rejected: writing into `match`/`game`/`move` while typing (see Context); a draft file in XDG
  (belongs to one library, needs a query, must travel with backups); a final save that closes
  the draft (no path for a correction found after saving); `Double/Take` as one Action (the
  cursor must stand between them); deriving the side at each replay (one deletion would flip
  every later Action); an `illegal` column on `move` (rule 5); the engine in JavaScript (a
  second copy of the rules).

## Guard

`pkg/blunderdb/transcript/*_test.go` (`inconsistency_test.go`, `replay_test.go`,
`gestures_test.go`, `mat_test.go`).
