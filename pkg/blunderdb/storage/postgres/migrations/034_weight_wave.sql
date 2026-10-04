-- Forward migration: completes the 2.31.0 wave with the weight wave of the
-- large-library plan (ADR-0071). The SQLite side lives in
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

-- Action labels become integer codes (domain.ActionCode, ADR-0071):
-- analysis.best_cube_action, move.move_type and move.cube_action. The fixed
-- codes below are domain.ActionLabels, index for index (the migration test
-- reads every one of them back); a label outside them is registered in
-- action_label from 1000 up. NULL stays NULL, '' is code 0. ALTER ... USING
-- admits no subquery, hence the column swap. Skipped once a column is INTEGER.
CREATE TABLE IF NOT EXISTS action_label (
    code  INTEGER PRIMARY KEY,
    label TEXT NOT NULL UNIQUE
);

DO $$
DECLARE
    c RECORD;
BEGIN
    CREATE TEMP TABLE action_label_fixed (code INTEGER PRIMARY KEY, label TEXT NOT NULL UNIQUE) ON COMMIT DROP;
    INSERT INTO action_label_fixed (code, label) VALUES (0, ''), (1, 'checker'), (2, 'cube'), (3, 'No Double'), (4, 'Double, Take'), (5, 'Double, Pass'), (6, 'Too good to double, pass'), (7, 'Too good to double, take'), (8, 'Double'), (9, 'Take'), (10, 'Pass'), (11, 'Drop'), (12, 'Beaver'), (13, 'Double/Take'), (14, 'Double/Pass'), (15, 'Double/Beaver'), (16, 'NoDouble'), (17, 'No double'), (18, 'Double, take'), (19, 'Double, pass'), (20, 'Too good to double'), (21, 'Too good'), (22, 'TG'), (23, 'Redouble'), (24, 'No Redouble'), (25, 'Redouble, Take'), (26, 'Redouble, Pass'), (27, 'Double / Take'), (28, 'Double / Pass'), (29, 'Double / Prendre'), (30, 'Double / Refuser'), (31, 'Double / Reject'), (32, 'No redouble'), (33, 'Redouble, take'), (34, 'Redouble, pass'), (35, 'Too good to redouble, pass'), (36, 'Too good to redouble, take'), (37, 'Double, Beaver'), (38, 'Double, beaver'), (39, 'Too Good'), (40, 'No Double, Take'), (41, 'No Double, Pass');
    FOR c IN SELECT * FROM (VALUES ('analysis', 'best_cube_action'), ('move', 'move_type'), ('move', 'cube_action')) AS t(tbl, col) LOOP
        IF (SELECT data_type FROM information_schema.columns
             WHERE table_schema = current_schema() AND table_name = c.tbl AND column_name = c.col) = 'text' THEN
            EXECUTE format(
                'INSERT INTO action_label (code, label)
                 SELECT (SELECT COALESCE(MAX(code), 999) FROM action_label) + ROW_NUMBER() OVER (ORDER BY v), v
                 FROM (SELECT DISTINCT %1$I AS v FROM %2$I WHERE %1$I IS NOT NULL) d
                 WHERE v NOT IN (SELECT label FROM action_label_fixed)
                   AND v NOT IN (SELECT label FROM action_label)', c.col, c.tbl);
            EXECUTE format('ALTER TABLE %I ADD COLUMN %I INTEGER', c.tbl, c.col || '_code');
            EXECUTE format(
                'UPDATE %2$I SET %3$I = COALESCE(
                     (SELECT f.code FROM action_label_fixed f WHERE f.label = %2$I.%1$I),
                     (SELECT l.code FROM action_label l WHERE l.label = %2$I.%1$I))
                 WHERE %1$I IS NOT NULL', c.col, c.tbl, c.col || '_code');
            EXECUTE format('ALTER TABLE %I DROP COLUMN %I', c.tbl, c.col);
            EXECUTE format('ALTER TABLE %I RENAME COLUMN %I TO %I', c.tbl, c.col || '_code', c.col);
        END IF;
    END LOOP;
    DROP TABLE action_label_fixed;
END $$;
