# The GUI's "write current position" goes through ingest, not through its own dedup

## Status

accepted

## Context

`importService.savePositionAndAnalysis` used to call `PositionExists` first and, when the
position was already known, skip `SavePosition` entirely and merge analysis and comment in
JavaScript. That gave the app two different notions of "same position": the backend's
Zobrist hash, and an O(n) full-scan JSON comparison in `PositionExists`.

It also silently broke `individually_imported` (ADR-0001): a user who imported a match and
then wrote one of its positions from the board would never reach the backend's upsert, so
the flag would never be set — in exactly the scenario the flag exists for.

## Decision

`:w` writes through the same `ingest.WritePosition` sink the file importers use. It
deduplicates by Zobrist, merges analyses, and sets `individually_imported`. `PositionExists`
is no longer on the write path.

Because match import appends one comment row per source note while the GUI treats a
Position as having a single comment blob, `WritePosition` takes an explicit comment
strategy: match import keeps `Append`, GUI write uses `MergeBlob` (the previous JS
substring-dedup + `\n\n` join, ported to Go). GUI comment behaviour is unchanged.

## Consequences

- One place writes a Position, so a new position attribute cannot be missed by one path.
- The underlying inconsistency remains and is recorded in CONTEXT.md: the `comment` table
  allows N rows per Position, but `LoadComment`/`SaveComment` read and edit whichever row
  comes back first. Fixing that is a separate piece of work.
