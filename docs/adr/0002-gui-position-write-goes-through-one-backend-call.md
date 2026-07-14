# Saving the board position is one backend call, not a frontend existence check

## Status

accepted

## Context

`importService.savePositionAndAnalysis` called `PositionExists` first and, when the
position was already stored, skipped `SavePosition` entirely, merging analysis and comment
in JavaScript against the id it had found.

That gave the app two different notions of "the same position": the store's Zobrist hash,
and an O(n) full-scan JSON comparison in `PositionExists`. Worse, it silently defeated
`individually_imported` (ADR-0001) in the case the flag exists for — import a match, then
save one of its positions from the board — because the write the flag rides on was never
reached.

## Decision

`:w` calls one backend method, `Database.SaveIndividualPosition`, which deduplicates on the
Zobrist hash, records the provenance, and returns `{id, existed}`. The frontend branches on
`existed` for its status message and keeps its analysis/comment merge exactly as it was.
`PositionExists` is off the write path.

## Considered options

- **Route `:w` through `ingest.WritePosition`**, the sink the file importers already use.
  Rejected on discovery: `ingest` *merges* an analysis into whatever is stored, while the
  GUI *replaces* it, and `ingest` appends a comment row while the GUI merges comments into
  a single blob. Sharing the sink would therefore have needed two behaviour switches
  (comment strategy, analysis strategy) — turning the shared write path into a
  configuration object, and changing observable GUI behaviour for analyses, which this
  change did not set out to do.
- **Add a `MarkIndividuallyImported(id)` call to the existing exists-branch.** Rejected: it
  keeps the O(n) scan and the second notion of identity, and leaves the trap in place for
  whoever adds the next position attribute.

## Consequences

- Provenance can no longer be missed by the most common way a position enters the database.
- The duplicated merge logic (analysis and comment) still lives in JavaScript. Unifying it
  with `ingest` means first reconciling merge-vs-replace for analyses, which is a
  behavioural decision of its own.
- CONTEXT.md records the underlying debt this exposed: the `comment` table allows N rows
  per position, but `LoadComment`/`SaveComment` read and edit whichever row comes back
  first.
