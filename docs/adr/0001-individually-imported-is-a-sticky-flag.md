# Individually-imported provenance is a sticky boolean, not a source enum

Status: accepted.

## Context

Positions deduplicate by Zobrist hash, so one row can be reached by several imports. A user
who imports a position on its own and later imports the match it came from ends up with a
single row, and needs to be able to find again the positions they brought in deliberately.

## Decision

1. `position` carries a boolean `individually_imported`, never part of the Zobrist hash.
2. Every path that writes a lone position sets it, including when the row already exists
   (`ON CONFLICT DO UPDATE`). Match import never sets it and never clears it.
3. The search exposes it as one binary filter (`i`).
4. `individually_imported` is a reason a position survives the orphan purge on match
   deletion, alongside Collection and Anki-card membership. A comment is not: importers
   attach the source file's notes as comments, so a comment is not evidence of user work.

## Consequences

- The flag reads "this position was, at some point, imported on its own" — order-independent.
- Databases predating the column were backfilled from the `move` heuristic, with its false
  positives (enrich-created positions) and irrecoverable false negatives.
- The retention predicate is stated in three places (`database/db_match.go`,
  `storage/sqlite/matches_sqlite.go`, `storage/postgres/matches_postgres.go`); they must not drift.
- Rejected: deriving it from "no `move` row" — inverts under a later match import, wrong for
  enrich-created positions.
- Rejected: a `source` enum — under dedup origin is a set; an enum makes the value depend on
  import order.
- Rejected: a user-toggleable marker — that is curation, already served by Collections and tags.

## Guard

`pkg/blunderdb/database/position_is_held_predicate_test.go`,
`pkg/blunderdb/database/delete_match_test.go`.
