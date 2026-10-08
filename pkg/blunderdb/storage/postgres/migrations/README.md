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
- `018_game_phase.sql` — `position.game_phase`, the derived phase label
  (issue #264, ADR-0035) and its index. No SQL backfill is possible: the
  classification reads the board out of the compact `state` encoding, so
  existing rows stay at 0 (unknown) until a repair pass rewrites them.
- `019_product_wave_2_19_0.sql` — the rest of the 2.19.0 wave: `comment.origin`
  (#263), `import_batch` + `match.import_batch_id` (#257) and `trash` (#285),
  with the `tenant_isolation` policy on the two new tenant-scoped tables.
  `origin` defaults to `'unknown'`, never `'user'` — see the file's header.
  Schema-visible: bumped `domain.DatabaseVersion` to 2.19.0.
- `021_transcription.sql` — the `transcription` table (#334, ADR-0045): the
  draft a match is typed into, one opaque JSON `document` carrying its own
  `format_version`, plus a nullable `match_id` pointing at the Match the draft
  has already produced (`ON DELETE SET NULL`, composite FK on
  `(tenant_id, id)`). Schema-visible: bumped `domain.DatabaseVersion` to
  2.21.0.
- `022_library_settings.sql` — `library_settings`, the tenant-scoped key/value
  table holding the library's error and blunder thresholds (ADR-0046). No
  `DatabaseVersion` bump: nothing changes on the SQLite side, where the same
  two rows live in the file's own `metadata` table. Creates no row — a tenant
  that set no threshold reads the defaults, which are the constants every
  consumer used before.
- `020_product_wave_2_20_0.sql` — the 2.20.0 wave: `position.max_cube` (#271),
  the session's cube ceiling as the XGID's log2 exponent, and
  `position.game_type` (#291), the derived plan of play. Neither joins the
  Zobrist hash (ADR-0028's conclusion, again) and `game_type` is 0 on every
  existing row until a repair pass reclassifies it — a derived label is
  recomputed, never backfilled by SQL. Schema-visible: bumped
  `domain.DatabaseVersion` to 2.20.0.
- `023_training_journal.sql` — `training_session` and `training_item`, the
  Training journal (#320, ADR-0040): two tables rather than a `metadata` key,
  because the per-number detail is the whole point of the journal and a blob
  that grows by one entry per revealed number is a register a table settles.
  Schema-visible: bumped `domain.DatabaseVersion` to 2.22.0.
- `024_anki_score_cards.sql` — `kind` and `key` on `anki_card` and
  `anki_review_log`, `position_id` becomes nullable on both, the
  `(deck_id, position_id)` uniqueness becomes `idx_anki_card_identity
  (deck_id, kind, key)` (#324, ADR-0042): a card asks about a position OR
  about a score, and a score card points at no position. Every existing card
  is backfilled to `position` with its id as key. Schema-visible: bumped
  `domain.DatabaseVersion` to 2.23.0.
- `025_direction.sql` — the 2.24.0 wave, the Direction of a Tournament
  (ADR-0047): `direction` and its append-only `direction_event` log, and
  `match.direction_match_id`, the Slot a Match fills in its Tournament's
  Direction (empty when none; at most one Match per Slot).
- `026_set_null_names_its_column.sql` — `match.import_batch_id` (019) and
  `transcription.match_id` (021) get the `SET NULL (column)` list that `001`
  and `017` already use. Without it a composite `ON DELETE SET NULL` nulls
  `tenant_id` too and the delete fails on its NOT NULL instead of unlinking
  the child. Constraint-only, like `006`/`008`/`010`/`017`: no
  `DatabaseVersion` bump.
- `027_rencontre.sql` — the Rencontre (ADR-0056): `rencontre`,
  `tournament.rencontre_id`, `direction_pair_member`.
- `028_game_winner_encoding.sql` — data only: every `game.winner` to the one
  encoding (1 = player 1, -1 = player 2, 0 = unfinished), by the statement
  SQLite's 2.25.0 → 2.26.0 step runs (`sqlshared.NormalizeGameWinnerSQL`,
  which documents the rule per source). Not idempotent on its own — a second
  pass would read normalized values as gnubg's — so it relies on
  `schema_migrations`, where its row is written in the same batch as the
  statement: either both land or neither does, and a replay finds the row and
  skips it. A fresh bootstrap holds no game to convert, so its run is a no-op.
  Lifts FORCEd RLS on the three tables it
  reads for the statement, since the migrating connection carries no tenant.
- `029_transcription_revision.sql` — `transcription.revision`, the version a
  transcription gesture names (ADR-0057 rule 4): incremented at every write,
  compared in the write's own statement, a column so that it is read without
  decoding the document. Existing rows start at 1. Schema-visible: bumped
  `domain.DatabaseVersion` to 2.27.0.
- `030_table_setting.sql` — the properties of a table and the rooms of an
  event (ADR-0058): `table_setting` (owned by a Rencontre or by a Tournament,
  never both; composite tenant keys, cascade with its owner) and
  `tournament.rencontre_rooms`, cleared on detachment. Schema-visible: bumped
  `domain.DatabaseVersion` to 2.28.0.
- `031_lesson.sql` — the Lesson (ADR-0066): `lesson` and `lesson_step`
  (composite tenant keys; a Step's collection and position are set to NULL
  when they are deleted, the Steps cascade with their Lesson). Schema-visible:
  bumped `domain.DatabaseVersion` to 2.29.0.
- `032_large_library_wave.sql` — the large-library wave: `analysis`
  provenance columns (`analysis_engine`, `analysis_depth`, `creation_date`),
  derived in Go after the forward chain by `backfillAnalysisProvenance`
  (`provenance_postgres.go`, the blob is compressed JSON, FORCE RLS lifted per
  batch transaction as here) and resumable on
  their NULLs; `position.match_date`, the earliest match reaching the
  position, backfilled set-based here and kept by the match store;
  `import_batch_file` (the per-file import journal), `player_alias` and
  `event_alias`; `match.dice_hash` and the source metadata of a match
  (ratings, experience, transcriber, Jacoby/Beaver, engine version); the
  `match_stats` table (per-match, per-seat tallies, empty until the stats
  store fills it); `training_item.position_id`/`answer`/`error_mp` and
  `comment.author`; and the index pruning decided on a 15.6 M-position
  library (`tasks/search-query-plans.txt`), `(game_phase, off_1)` replacing
  the phase index. Schema-visible: bumped
  `domain.DatabaseVersion` to 2.30.0.
- `033_met_progress_move_error.sql` — the 2.31.0 wave: `match_equity_table`
  (the tenant's imported match equity tables, at most one current, ADR-0068)
  and `analysis.met_digest`; `lesson_progress`, the Steps a student marked
  done (ADR-0069), with the `(tenant_id, id)` key on `lesson_step` its
  composite foreign key needs; `move.error_mp`, added NULL so the migration
  rewrites nothing, scored afterwards by the resumable
  `MatchStore.ScoreMoves`. Schema-visible: bumped `domain.DatabaseVersion`
  to 2.31.0.
- `034_weight_wave.sql` — completes the 2.31.0 wave (ADR-0071), with no
  bump of its own: `analysis.met_digest` gives way to `analysis.met_id`
  (NULL = the built-in table); `idx_analysis_engine` and `idx_analysis_depth`
  dropped for the partial `idx_analysis_provenance_pending`;
  `match.comment_author`; `position.state` becomes the BYTEA of
  `engine.EncodeBoardState`; `analysis.best_cube_action`, `move.move_type` and
  `move.cube_action` become integer codes (`domain.ActionLabels`), a label
  outside them registered per tenant in `action_label` from 1000 up. Row-level security, when FORCEd, is lifted
  for the data-moving statements and restored at the end.
- `035_study_mark.sql` — `study_mark`, one row per position the user marked
  studied; not a retention reason. Part of the 2.31.0 schema.
- `036_match_stats_cells.sql` — `match_stats_cell` and `match_stats_position`,
  the per-match breakdowns beside `match_stats` (kinds and arithmetic in
  `sqlshared/match_stats_cells.go`), cascading with their seat's
  `match_stats` row. Part of the 2.31.0 schema.
- `037_duel.sql` — the 2.32.0 wave (ADR-0072): `duel`, the draft of a match
  played here (an opaque document, and the dice seed in a column written once),
  and `match_origin`, how a Match played here came to be, cascading with its
  Match. Schema-visible: bumped `domain.DatabaseVersion` to 2.32.0.
- `038_decision_time.sql` — the 2.33.0 wave (ADR-0073): `move.decision_ms` and
  `move.cube_decision_ms`, the time a Duel measured for the decision, NULL for
  unknown; `match_origin.lost_on_time` renamed `over_time`, the player whose
  reserve ran out. Schema-visible: bumped `domain.DatabaseVersion` to 2.33.0.
- `039_bot_engine.sql` — the 2.34.0 wave (ADR-0072 rule 10):
  `match_origin.bot_engine`, the gammonNet tag whose policy a Bot played, ''
  when none did or for an origin stored before. Schema-visible: bumped
  `domain.DatabaseVersion` to 2.34.0.
- `040_duel_external_sides.sql` — the 2.35.0 wave (ADR-0072 rules 8 and 10):
  `match_origin.declared_bots`, the Bots external Sides declared, as JSON, ''
  when none did; `match_origin.contributions`, the Sides' contributions to a
  combined seed, as JSON, '' when there was none. Schema-visible: bumped `domain.DatabaseVersion` to 2.35.0.
- `041_duel_open.sql` — the 2.36.0 wave (ADR-0072 rule 10): `duel.is_open`,
  1 while the Duel is being played, so several can be open and every daemon on
  the database sees the same ones. Schema-visible: bumped
  `domain.DatabaseVersion` to 2.36.0.
- `042_pr_xg_rules.sql` — the 2.37.0 wave: `analysis.is_forced`,
  `is_close_cube` and `best_move_equity_error` take the rules XG counts a
  Performance Rating by (`engine.IsForcedChecker`, `engine.ComputeIsCloseCube`,
  NULL for a played move no candidate names). No SQL backfill is possible —
  the columns project the compressed blob and the position's legal plays — so
  the existing rows keep their old values until a repair pass
  (`RepairDenormalisedColumns`) rewrites them; `match_stats` is dropped.
  Schema-visible: bumped `domain.DatabaseVersion` to 2.37.0.
