# The trash is a snapshot table, not a deleted_at column

Status: accepted.
See also: ADR-0007

## Context

Deletions (position, collection, comment, Anki card) were final on confirmation. A
`deleted_at` column would oblige every read to filter it — some fifty search filters, both
statistics backends, the three-place retention predicate, the Anki scheduler, the export —
and one that forgets fails silently. The UNIQUE `idx_position_zobrist` makes it worse: a
soft-deleted row keeps its hash and collides with the re-import of its match.

## Decision

1. **Deleting stays a real DELETE, preceded by a snapshot into `trash`**: one row per deleted
   thing — `kind` (position, collection, comment, anki_card), `label`, `payload` (JSON, what
   is needed to put it back), `deleted_at`.
2. **No live query changes.** Only the trash panel and the purge read `trash`.
3. **Restoring a position re-saves it through `SavePosition`**: same row if its Zobrist hash is
   free, merged into the existing row otherwise.
4. **Retention is 30 days, purged by `blunderdb vacuum`.** Nothing purges on open.

## Consequences

- Undo for a new gesture is snapshot-then-delete; a gesture that does not snapshot behaves as
  before, with no half-state.
- Restore is "put this back", not "undo the transaction": what cascaded by foreign key comes
  back only as far as the payload of that kind holds it.
- The trash travels with the file but never in an export (`trash` is not in the ADR-0007
  allow-list).
- Snapshots cost disk until vacuum; `blunderdb verify` reports the count.
- Rejected: a `deleted_at` column — every reader must learn it, and it breaks Zobrist dedup.

## Guard

`pkg/blunderdb/database/db_trash_test.go`.
