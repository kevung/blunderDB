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

-- Who signs match.comment: a comment an XG file carries is signed by its
-- transcriber, a comment written in blunderDB by its author.
ALTER TABLE match ADD COLUMN IF NOT EXISTS comment_author TEXT NOT NULL DEFAULT '';

-- position.state becomes the 28 signed bytes of engine.EncodeBoardState in
-- place of the compact JSON array; a legacy full-Position JSON state keeps its
-- text as bytes, which the decoder still reads. ALTER ... USING admits no
-- subquery, hence the column swap. Skipped once the column is BYTEA.
DO $$ BEGIN
    IF (SELECT data_type FROM information_schema.columns
         WHERE table_schema = current_schema() AND table_name = 'position' AND column_name = 'state') = 'text' THEN
        ALTER TABLE position ADD COLUMN state_bin BYTEA;
        UPDATE position SET state_bin = CASE
            WHEN left(state, 1) = '[' AND jsonb_array_length(state::jsonb) = 28 THEN
                (SELECT decode(string_agg(lpad(to_hex((v::int + 256) % 256), 2, '0'), '' ORDER BY o), 'hex')
                   FROM jsonb_array_elements_text(state::jsonb) WITH ORDINALITY AS t(v, o))
            ELSE convert_to(state, 'UTF8') END;
        ALTER TABLE position DROP COLUMN state;
        ALTER TABLE position RENAME COLUMN state_bin TO state;
        ALTER TABLE position ALTER COLUMN state SET NOT NULL;
    END IF;
END $$;
