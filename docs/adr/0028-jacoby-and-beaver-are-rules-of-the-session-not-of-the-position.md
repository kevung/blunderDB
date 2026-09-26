# Jacoby and beaver are rules of the session, not of the position

Status: accepted.
See also: ADR-0001

## Context

The Zobrist hash is what deduplicates positions across imports. Only one door sets
`has_jacoby`/`has_beaver`: `domain.DecodeXGID`, from XGID field 7 in money play. No file
importer sets them, and the XG and gnuBG parsers do not even surface them (BGF has them only
per match). Folding them into the hash splits one money position into two rows — XGID paste
vs `.xg` import — with analyses on one and the comment on the other.

## Decision

1. `has_jacoby` and `has_beaver` are not part of the Zobrist hash. They remain `position`
   columns, written by the XGID decoder, carried by export/import and rendered on the board.
2. Their two keys stay drawn from the key stream in `engine.init` (`zobristRetiredJacoby`,
   `zobristRetiredBeaver`) and are never folded: removing the draws would shift every later
   key and rehash every database.
3. Converting a stored hash is XORing the retired keys back out (`engine.RetiredFlagDelta`).
   Rows that meet on one hash are merged into the **oldest** through `mergePositionInto`: the
   keeper inherits moves, comments, collection memberships and Anki cards; sticky marks
   (ADR-0001, ADR-0006) are raised, never lowered; an analysis follows only onto a keeper
   that has none.

## Consequences

- The stored flags are those of the last write, like every field outside the identity.
- The PostgreSQL conversion writes the two keys as literals
  (`storage/postgres/migrations/014_zobrist_without_rule_flags.sql`); it is the one
  non-idempotent migration of that chain (XOR is its own inverse) and `schema_migrations`
  prevents a replay.
- Rejected: filling the flags in every importer — the files do not carry them, and two money
  sessions with different settings would still store one board twice.
- Rejected: hashing them only in money play — identity conditional on another field.
- Rejected: a second deduplication pass — a second notion of "same position".
- Rejected: dropping the columns — they are shown and round-trip through XGID.

## Guard

`TestZobristIgnoresJacobyAndBeaver` in `pkg/blunderdb/engine/zobrist_test.go`;
`pkg/blunderdb/storage/postgres/zobrist_retired_keys_test.go`.
