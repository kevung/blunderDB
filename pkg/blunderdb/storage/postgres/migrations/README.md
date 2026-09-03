# PostgreSQL migrations

The PostgreSQL backend tracks its **own** forward migration chain, independent
of the SQLite one.

## Why no historical port

The SQLite backend carries 15 historical migrations (`db_migration.go`) that
upgrade pre-2.x databases written by older blunderDB releases. None of those
old databases exist as PostgreSQL databases — PostgreSQL is a new backend
introduced for the `serve` mode. Porting the historical chain would be dead
code.

So PostgreSQL **starts fresh at the terminal SQLite schema, v2.7.0**:

- `001_initial_v2_7_0.sql` — the complete v2.7.0 schema, multi-tenant.

## Forward chain

New schema changes are added as `NNN_description.sql` files, applied in
numeric order by `migrateForward` (see `migrate_postgres.go`). On every `Open`
/ `Migrate`, each file beyond the `001` baseline that is not yet recorded in the
`schema_migrations` table is applied (simple-protocol batch) and recorded. Make
every migration **idempotent** (`ADD COLUMN IF NOT EXISTS`,
`CREATE INDEX IF NOT EXISTS`, set-based backfills) so that applying it to a
freshly bootstrapped database — whose `001` baseline already contains the change
— is a harmless no-op.

The whole sequence — `Migrate`'s freshness probe, `bootstrap` (fresh databases
only) and `migrateForward` — runs on one connection held under a session-level
`pg_advisory_lock`, so two processes calling `Migrate` at once (two daemon
replicas starting together, or the daemon racing a `blunderdb migrate`
invocation) serialize instead of racing the same DDL (#231).

A migration file must **not** write `database_version` itself. Until #231 each
one stamped its own intermediate value — `009` set `2.15.0`, then `013` set
`2.17.0` — so a process interrupted between the two left the database at its
true (newer) schema while `metadata` still named `2.15.0`; `/readyz` then
either passed on a half-migrated database or failed on a fully-migrated one,
depending on exactly where the interruption landed. `Migrate` now writes
`database_version` from `domain.DatabaseVersion` exactly once, after bootstrap
and every forward migration have both already succeeded — the version a
reader sees is either the old one (nothing ran yet) or the new one (the whole
chain committed), never a value in between.

- `002_is_cube_response.sql` — `position.is_cube_response` column + index, with a
  take/pass backfill from `move.cube_action` (mirrors
  `engine.IsResponseCubeAction`).
- `006_comment_position_index.sql` — `comment(tenant_id, position_id)` index for
  the comment-presence search filter (`co` / `xco`).
- `007_flagged.sql` — `position.flagged` column + index, the source-tool study
  mark (docs/adr/0006). No backfill is possible: the flag exists only in the
  source files, never in an already-imported database.
- `008_win_gammon_covering_index.sql` — extends the win/gammon combo search
  index with a trailing `position_id` column so the query's `p.id IN (SELECT
  position_id FROM analysis WHERE …)` subquery is answered from the index
  alone (fiche-05 T3). Index-only, like `006`.
- `009_luck_mp.sql` — `move.luck_mp` column, the luck of a roll in signed
  millipoints (docs/adr/0010). NULLable with no default: NULL means unknown,
  which is not the same as a neutral roll. No backfill is possible — luck
  exists only in the source files, never in an already-imported database.
- `010_search_range_indexes.sql` — the eight single-column search-range
  indexes SQLite gained in fiche-05 (`back_checkers_1/2`, `pip_1`,
  `no_contact`, `player1_backgammon_rate`, `player2_win/gammon/backgammon_rate`),
  tenant_id leading. Index-only, like `006` and `008`.
  `index_parity_test.go` (no Docker) keeps the two backends' `idx_*` name sets
  aligned from here on.
- `011_exclude_position.sql` — `filter_library.exclude_position` and
  `search_history.exclude_position`, the "Sauf" structure SQLite gained in
  2.8.0 and this backend never received. A catch-up, not a new schema
  version: `domain.DatabaseVersion` is left alone.
- `012_anki_session_limit.sql` — `anki_deck.session_limit` column, how many
  cards one review sitting serves per deck (ADR-0026 rule 2). Nullable with
  no default: NULL is "no limit" and is what every pre-existing deck keeps.
  Schema-visible: bumped `domain.DatabaseVersion` to 2.16.0.
- `013_session_state.sql` — `session_state(tenant_id, key, value)`, the UI
  session state that lived in `metadata` as `<scope>:session_*` rows until
  2.16.0 and was readable by every tenant through `metadata.load` (#156).
  Moves the rows of every integer-named tenant, drops the rest (named
  tenants no longer exist, ADR-0005), and installs the `tenant_isolation`
  policy on the new table when the database already enforces RLS.
  Schema-visible: bumped `domain.DatabaseVersion` to 2.17.0.
- `014_zobrist_without_rule_flags.sql` — the Jacoby and beaver flags leave the
  position identity (ADR-0028, #171). A XOR undoes the fold with the two retired
  Zobrist keys, written as literals and pinned to the engine by
  `zobrist_retired_keys_test.go`; rows the rehash brings onto one hash were
  always the same position and are merged onto the oldest of them. The **one
  migration in this chain that is not idempotent** — XOR is its own inverse, so
  replaying it would undo the conversion; `schema_migrations` is what stops
  that, and a freshly bootstrapped database carries no pre-2.18.0 hash to
  convert. Schema-visible: bumped `domain.DatabaseVersion` to 2.18.0.
- `015_one_analysis_per_position.sql` — `analysis(position_id)` becomes UNIQUE
  (#173), which is what makes `analyses_postgres.go`'s upsert legal: an
  `ON CONFLICT` target must name a unique constraint. Deduplicates first,
  keeping the highest id per position. Also adds the range CHECKs of `001`
  as `NOT VALID` so a database whose history predates the rule still opens —
  `blunderdb verify` names the offending rows. 014, 015 and 016 are one
  2.18.0 wave: `domain.DatabaseVersion` moved once, with `014`.
- `016_review_log_foreign_keys.sql` — `anki_review_log.deck_id` and
  `.position_id` become real foreign keys (#185), composite (`tenant_id`, …)
  from the start (#235; `017` promotes every other tenant-scoped foreign key
  the same way, and this table's own `card_id` follows suit there too — see
  `017`'s entry below). Added `NOT VALID`: every row written from here on is
  governed, the rows already there are not scanned, and a database carrying a
  dangling journal row keeps opening — purging a user's review history is not
  a migration's decision.
- `017_composite_tenant_fk.sql` — every foreign key between two tenant-scoped
  tables becomes composite (`tenant_id`, `parent_id`) `REFERENCES parent
  (tenant_id, id)` instead of `parent_id REFERENCES parent (id)` alone
  (#235): `id` is already globally unique (`BIGSERIAL`), so the single-column
  form was never wrong about *which* row a foreign key points to, only silent
  about whether that row belongs to the same tenant. Constraint-only, like
  `006`/`008`/`010`: `domain.DatabaseVersion` is left alone.

When you add a migration, also fold the change into `001_initial_v2_7_0.sql`
(so fresh databases get it directly), bump `domain.DatabaseVersion` if
schema-visible, and extend the migration test — but do **not** have the
migration write `database_version` itself (see above): `Migrate` is the one
place that happens, once, from the Go constant.

An **index-only** migration is the exception: it changes no column, no table and
no data, so nothing is schema-visible. It needs no `domain.DatabaseVersion`
bump — recording one with no matching schema change would desynchronise the
two — and no migration-test entry. `006` is one such migration.

## Multi-tenancy

Every domain table has `tenant_id BIGINT NOT NULL`. The application filters by
`tenant_id` on every query; this is mandatory regardless of whether the
optional Row-Level Security policies (see `../RLS.md`) are enabled.

The `metadata` table is database-level infrastructure (it holds the schema
version and the issuance document) and is **not** tenant-scoped — which is
why it must hold no per-tenant data: the daemon exposes it read-only
(`metadata.version`), and the session state that used to sit in it moved to
`session_state` in `013` (#156).
