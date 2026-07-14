# Individually-imported provenance is a sticky boolean, not a source enum

## Status

accepted

## Context

Positions deduplicate by Zobrist hash, so one row can be reached by several imports. A
user who imports a position on its own and later imports the match it came from ends up
with a single row — and, before this decision, no way to tell that they had ever taken an
interest in it. The position was lost among the thousands the match brought in.

## Decision

`position` carries an immutable boolean `individually_imported`. Every import path that
writes a lone position sets it — including when the row already exists (`ON CONFLICT DO
UPDATE`), so the flag survives a match having been imported first. Match import never sets
it and never clears it (`ON CONFLICT DO NOTHING`). The search layer exposes it as one
binary filter (`i`).

## Considered options

- **Derive it** as `NOT EXISTS (SELECT 1 FROM move WHERE position_id = p.id)` — free, no
  migration. Rejected: it inverts under a later match import (the exact scenario the
  feature exists to serve), and it is wrong for the positions created by a cross-format
  "enrich" import, which are match-sourced yet have no `move` row (`ingest/match.go:128`).
- **A `source` enum column** (`'match' | 'individual'`). Rejected: under deduplication a
  position's origin is a *set*, not a scalar. An enum forces one import to overwrite the
  other, making the value depend on the order the user happened to import their files in —
  a semantics that cannot be explained to a user.
- **A user-toggleable marker.** Rejected: it would answer "which positions do I care
  about", which is a different question, already served by Collections and tags. Merging
  curation into provenance would corrupt the provenance and leave the two concepts
  permanently inseparable.

## Consequences

- The flag reads as "this position was, at some point, imported on its own" — order-independent, and true.
- Existing databases are backfilled once from the `move` heuristic, inheriting its known
  false positives (enrich-created positions) and irrecoverable false negatives (a position
  individually imported before a match that also contained it). The information to do
  better does not exist in those databases.
- `individually_imported` joins Collection membership and Anki-card membership as a reason a
  Position survives the orphan purge on match deletion. Without this, deleting a match would
  silently delete the very positions the filter exists to surface. A *comment* was considered
  as a further reason and rejected: match importers attach the source file's per-move notes as
  comments (`ingest/xg.go`), so a comment is not evidence of user work, and holding on it
  would keep an entire annotated match alive after the user deleted it.
- The retention predicate is stated in three places — both Storage backends and the `Database`
  wrapper, which is the copy the GUI and the CLI actually run. They must not drift.
