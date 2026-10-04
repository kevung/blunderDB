-- Forward migration: completes the 2.31.0 wave with the weight wave of the
-- large-library plan (ADR-0070). The SQLite side lives in
-- migrate_2_30_0_to_2_31_0; 2.31.0 was never published, so this does not bump
-- domain.DatabaseVersion.
--
--   * analysis.met_digest TEXT gives way to analysis.met_id, the id of the
--     match_equity_table row (NULL = the built-in table). No code wrote the
--     digest column, so nothing is carried over.
--   * idx_analysis_engine and idx_analysis_depth are dropped: their filters
--     run inside a correlated lookup on position_id. The provenance backfill
--     finds its pending rows through a partial index that empties once the
--     pass is over.
--   * The dates (position.match_date, analysis.creation_date) stay
--     TIMESTAMPTZ: PostgreSQL already stores that type as an 8-byte integer
--     in UTC, the representation SQLite now uses in Unix seconds.

ALTER TABLE analysis DROP COLUMN IF EXISTS met_digest;
ALTER TABLE analysis ADD COLUMN IF NOT EXISTS met_id BIGINT REFERENCES match_equity_table (id);

DROP INDEX IF EXISTS idx_analysis_engine;
DROP INDEX IF EXISTS idx_analysis_depth;
CREATE INDEX IF NOT EXISTS idx_analysis_provenance_pending ON analysis (id) WHERE analysis_engine IS NULL;
