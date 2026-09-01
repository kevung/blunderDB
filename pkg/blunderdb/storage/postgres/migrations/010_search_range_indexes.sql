-- Forward migration: the eight search-range indexes the SQLite backend gained
-- in fiche-05 (schema_sqlite.go / db_schema.go, "Range filters that previously
-- had no supporting index") and that this backend never received. Each one
-- backs a single-column range filter of the search panel — back checkers
-- (k/K), player pip count (P), no contact, and the four rate filters on the
-- opponent's side plus player 1's backgammon rate (b/W/G/B) — that was until
-- now a full position or analysis scan per search.
--
-- Same columns as SQLite, with tenant_id leading every index: every query in
-- this backend is tenant-scoped, and a leading tenant_id lets the planner
-- satisfy the always-present `WHERE tenant_id = $1` from the index
-- (see the index block of 001_initial_v2_7_0.sql).
--
-- idx_position_no_contact is partial, like its SQLite twin (`WHERE no_contact
-- = 1`). Its predicate is spelled `no_contact IS TRUE`, not the bare
-- `no_contact` the other partial indexes of this backend use, because the
-- search filter is written `p.no_contact IS TRUE` (search_postgres.go) — the
-- column is nullable — and PostgreSQL does not prove that `x IS TRUE` implies
-- `x`: a `WHERE no_contact` index is simply never chosen for that query
-- (verified with EXPLAIN on postgres:16, enable_seqscan off — still a
-- Seq Scan). With the predicate written as the query writes it, the match is
-- textual and the index is used.
--
-- Index-only: no column, no table, no data changes. Nothing here is
-- schema-visible, so this migration deliberately does NOT bump
-- database_version in metadata, and domain.DatabaseVersion is left alone —
-- recording a version with no counterpart in the Go constant would
-- desynchronise the two (see migrations/README.md).
--
-- Idempotent, so it is safe on a fresh database whose 001 baseline already
-- creates the indexes.

CREATE INDEX IF NOT EXISTS idx_position_back_checkers_1 ON position (tenant_id, back_checkers_1);
CREATE INDEX IF NOT EXISTS idx_position_back_checkers_2 ON position (tenant_id, back_checkers_2);
CREATE INDEX IF NOT EXISTS idx_position_pip_1           ON position (tenant_id, pip_1);
CREATE INDEX IF NOT EXISTS idx_position_no_contact      ON position (tenant_id) WHERE no_contact IS TRUE;
CREATE INDEX IF NOT EXISTS idx_analysis_backgammon1     ON analysis (tenant_id, player1_backgammon_rate);
CREATE INDEX IF NOT EXISTS idx_analysis_win2            ON analysis (tenant_id, player2_win_rate);
CREATE INDEX IF NOT EXISTS idx_analysis_gammon2         ON analysis (tenant_id, player2_gammon_rate);
CREATE INDEX IF NOT EXISTS idx_analysis_backgammon2     ON analysis (tenant_id, player2_backgammon_rate);
