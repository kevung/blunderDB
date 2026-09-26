# Error and blunder thresholds are library settings

Status: accepted.
See also: ADR-0005, ADR-0007

## Context

The blunder line (100 millipoints) and the study queue's line (50) were fixed constants in
different places, and "error" meant any non-zero cost. Users want both lines theirs. A
per-machine setting would make one file count different blunders on two computers, and the
CLI, which never reads the XDG config, would contradict the desktop. On the daemon `metadata`
is one table for every tenant, outside RLS.

## Decision

1. **Two nested thresholds.** An Error is a decision whose cost reaches the error threshold; a
   Blunder is an Error whose cost reaches the blunder threshold. Error ≤ blunder is enforced on
   write. Defaults 50 and 100; XG (20/80) and gnubg (40/80) are named presets, never the truth.
2. **They are Library settings**, behind one accessor of the `storage` contract. SQLite keeps
   them in `metadata`; PostgreSQL in a tenant-scoped key/value table under RLS (migration
   `022`, no `DatabaseVersion` bump; a tenant with no row reads the defaults). `migrate`
   copies them, `blunderdb edit` sets them, `info` prints them, the daemon exposes a
   tenant-scoped `/v1` route. They are **not** in `issuance.CarriedMetadataKeys`.
3. **One reader.** The statistics, players' rows, status-bar counter and study queue all read
   the accessor; no constant survives. The counter scores a position played several ways by its
   largest cost, as `E>x` does, so it counts what the search it opens returns.
4. **Where they are set**: a *Library* tab of the configuration window, greyed without an open
   database, which also holds Vacuum and Repair. Entered in equity with three decimals, stored
   in millipoints; the counter's link pre-fills the current value (`s E>80`).

## Consequences

- The import report's JSON reason `blunder` keeps its string (CLI contract), documented as
  "error, in the library's sense".
- The error histogram keeps fixed magnitudes: it shows a distribution, not categories.
- Rejected: per machine, in the config file — inconsistent across machines, CLI cannot follow.
- Rejected: daemon defaults with no tenant setting — a migrated library would count differently.
- Rejected: a third category beside "Errors" — three words for two lines.
- Rejected: a search token resolving to the current threshold — a saved `E>80` says exactly
  what it returns; the token can be added later without undoing anything.
- Rejected: keeping the study queue's own constant — three lines for two words.

## Guard

`TestExport_ThresholdsDoNotTravel` in `pkg/blunderdb/database/collection_test.go`;
`pkg/blunderdb/database/stats_storage_parity_test.go`.
