# Saving the board position is one backend call, not a frontend existence check

Status: accepted.
See also: ADR-0001

## Context

A frontend that checks `PositionExists` and skips the write when the position is stored has
two notions of identity (Zobrist hash vs an O(n) JSON scan) and never reaches the write that
sets `individually_imported` — in exactly the case the flag exists for (import a match, then
save one of its positions from the board).

## Decision

`:w` calls one backend method, `Database.SaveIndividualPosition`, which deduplicates on the
Zobrist hash, records the provenance, and returns `{id, existed}`. The frontend branches on
`existed` only for its status message and keeps its analysis/comment merge. `PositionExists`
is off the write path.

## Consequences

- Provenance cannot be missed by the most common way a position enters the database.
- The analysis/comment merge still lives in JavaScript: unifying it with `ingest` first
  requires deciding merge-vs-replace for analyses (the GUI replaces, `ingest` merges).
- The `comment` table allows N rows per position while `LoadComment`/`SaveComment` edit
  whichever comes first (recorded in `CONTEXT.md`).
- Rejected: routing `:w` through `ingest.WritePosition` — needs two behaviour switches and
  changes the GUI's analysis semantics.
- Rejected: adding `MarkIndividuallyImported(id)` to the exists-branch — keeps the O(n) scan
  and the second notion of identity.

## Guard

`internal/cli/parity_test.go`, `pkg/blunderdb/database/db_import_db_test.go`.
