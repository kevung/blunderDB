# Jacoby and beaver are rules of the session, not of the position

## Status

accepted — 2026-09-03. Amends [0001](0001-individually-imported-is-a-sticky-flag.md):
that record states what provenance is not part of the Zobrist identity; this one
removes two more fields from it. Schema 2.18.0 carries the conversion.

## Context

A position is identified by its Zobrist hash, and that hash is what deduplicates
positions across imports (invariant no. 1). Until schema 2.18.0 the hash folded in
`has_jacoby` and `has_beaver` alongside the board, the dice, the cube and the score.

Only one door sets those two columns: `domain.DecodeXGID` reads field 7 of an XGID as
a Jacoby/beaver bitmask in money play (`domain/xgid.go:130-145`). Every file importer
leaves them at 0 — `ingest/xgmap.go`, `ingest/gnubgmap.go` and `ingest/bgfmap.go`
build their `domain.Position` without ever naming the fields, and the parsers behind
two of them do not surface the information at all (`gnubgparser.MatchMetadata` has no
such field; `xgparser` none either).

The consequence is a split identity. The same money position pasted from an XGID with
Jacoby on, and imported from a `.xg` file, hashes to two different values and lands on
two rows. The user sees the position twice, its analyses on one row and their comment
on the other, and no filter reunites them. Deduplication — the reason the hash exists —
silently stops working on exactly the positions where money play makes Jacoby relevant.

## Decision

**`has_jacoby` and `has_beaver` leave the Zobrist hash.** They remain columns of
`position`, are still written by the XGID decoder, still travel through export and
import, and are still what the board renders. They simply no longer decide which row a
position is.

The two keys stay drawn from the key stream in `engine.init` (`zobristRetiredJacoby`,
`zobristRetiredBeaver`): removing the draws would shift every key allocated after them
and change the hash of every position ever stored. Drawn but not folded, a position
that never carried either flag — nearly all of them — keeps the hash it has.

Schema 2.18.0 converts the rest. A Zobrist hash is a XOR of keys, so undoing a fold is
XORing the same key back in (`engine.RetiredFlagDelta`): the migration reads the two
flag columns and needs no board, no state and no decode. Rows the conversion brings
onto one hash were the same position all along and are merged, the **oldest** row
kept, through the same `mergePositionInto` that folds the duplicates
`repairPositionsWithoutScalars` finds.

## Considered options

- **Fill the flags in every importer instead.** Rejected: the information is not in the
  files. XG and gnuBG's parsers expose no Jacoby or beaver setting, and BGF exposes a
  *match-level* `useJacoby`/`useBeaver` — a property of the session, which is precisely
  the argument for taking them out of the position's identity rather than into it. Even
  filled everywhere, one user's money session played with Jacoby and another's played
  without would still store the same board twice.
- **Hash them only in money play** (where they mean something) **and not at a match
  score.** Rejected: it makes the identity conditional on another field, so a position
  whose score is edited changes hash for a reason no one can predict, and it does not
  solve the case it exists for — two money sessions with different settings.
- **Leave the hash alone and deduplicate in a second pass.** Rejected: a second notion
  of "same position", maintained beside the first, is the bug not the fix. There is one
  identity and it is the hash.
- **Drop the columns.** Rejected: they are shown on the board and round-trip through
  XGID. What is being taken away is their vote on identity, not the fact.

## Consequences

- Two rows can merge on upgrade. The kept row is the oldest, it inherits the
  duplicate's moves, comments, collection memberships and Anki cards, and the sticky
  marks (`individually_imported`, `flagged` — ADR-0001, ADR-0006) are raised on it and
  never lowered. An analysis follows only onto a keeper that has none.
- A position saved with Jacoby set and later reached without it now updates the same
  row, so the *stored* flags are those of the last write. That is the same "last writer
  wins" the dice, the cube owner and the score have always had for fields outside the
  identity — and unlike them, these two are a property of the session, so no reading of
  the position is falsified by it.
- The PostgreSQL backend converts with the same XOR, written as two literals in
  `migrations/014_zobrist_without_rule_flags.sql` and pinned to the engine by
  `zobrist_retired_keys_test.go`. Migration 014 is the one non-idempotent migration of
  that chain: XOR is its own inverse, and `schema_migrations` is what stops a replay.
- `TestZobristIgnoresJacobyAndBeaver` guards the decision the way
  `TestZobristIgnoresIndividuallyImported` guards ADR-0001.
