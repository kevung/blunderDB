# Source-tool study marks are a sticky position property, not an auto-filled collection

Status: accepted.
See also: ADR-0001

## Context

eXtreme Gammon lets a player flag a decision while reviewing a match; the mark is stored per
move in the `.xg` file (`MoveEntry.Flagged`, `CubeEntry.FlaggedDouble`). Neither GnuBG nor
BGBlitz records an equivalent. Dropping the mark on import loses the user's study list.

## Decision

1. `position` carries a boolean `flagged`, written only by the importer, ORed into the stored
   value like `individually_imported`, never part of the Zobrist hash. No gesture inside
   blunderDB sets or clears it.
2. The search exposes it as one binary filter (`fl`).
3. A flagged cube decision marks both positions derived from it (double and take/pass).
4. `WriteMatch` applies flags even to an exact duplicate match it otherwise skips: the match
   hash comes from the play, not the file, so a newly flagged file is not a new match.
5. A flagged position survives the orphan purge on match deletion (see ADR-0001 rule 4 and
   its three-place retention predicate).

## Consequences

- The mark reads "flagged at some point in the source tool" — order-independent.
- No backfill is possible: existing positions gain the mark when their match is re-imported,
  which is why rule 4 exists.
- A flag set by mistake is permanent; a transient list is a Collection.
- `ingest` re-reads raw XG segments to recover the marks (the lightweight `xgparser.Match`
  drops them); the key mapping must mirror `ParseXG`'s record→Move numbering.
- Rejected: an auto-filled Collection — a Collection is assembled by hand, and importer-written
  membership would change which positions survive a match deletion.
- Rejected: storing it on `move` — an `EXISTS` on every search, and standalone positions
  could not be marked.
- Rejected: recomputing on re-import — one file could clear another's mark under dedup.
- Rejected: including it in the match hash — would re-import the match as a duplicate.
- Rejected: a settings toggle — reading a field the file carries needs no switch.

## Guard

`pkg/blunderdb/ingest/xg_flagged_test.go`,
`pkg/blunderdb/database/position_is_held_predicate_test.go`.
