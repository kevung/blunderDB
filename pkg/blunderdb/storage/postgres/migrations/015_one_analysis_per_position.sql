-- Forward migration: one analysis per position, and range constraints on what
-- the columns are allowed to hold (issue #173, schema 2.18.0).
--
-- analysisStore.Save used to SELECT an existing row and then INSERT or UPDATE.
-- Two saves racing on the same position both read "no row" and both inserted,
-- and Load — a plain `WHERE position_id = $1` — then returned whichever row the
-- planner reached first: a position could show an analysis that had been
-- superseded, with nothing visible to say so. Save is a single upsert now, and
-- an ON CONFLICT target must name a UNIQUE constraint, so the index is not
-- documentation here — it is what makes the statement legal.
--
-- Idempotent: the delete finds nothing on a second run, the index is
-- IF NOT EXISTS, and each CHECK is added only if no constraint of that name
-- exists (the names are the ones 001 declares, so a freshly bootstrapped
-- database skips them all).

-- Keep the HIGHEST id per position: the last row written is the one Save meant
-- to leave.
DELETE FROM analysis a USING analysis b
WHERE a.position_id = b.position_id AND a.id < b.id;

DROP INDEX IF EXISTS idx_analysis_position;
CREATE UNIQUE INDEX IF NOT EXISTS idx_analysis_position ON analysis (position_id);

-- position.zobrist_hash stays NULLABLE, here as in the SQLite backend, and for
-- the SQLite backend's reasons: EnsureSchema cannot ALTER a missing column into
-- existence with a NOT NULL and no default, repairPositionsWithoutScalars needs
-- to find rows with a NULL hash in order to repair them, and a constraint the
-- two backends state differently is worse than one they both report. The rule
-- lives in `blunderdb verify` (database/db_verify.go, CheckConstraints), which
-- can tell the truth about every database rather than about the ones created
-- since.

-- Range constraints, added NOT VALID: new and updated rows are checked, rows
-- already there are left alone. That is the same bargain the SQLite backend
-- strikes (it cannot add a CHECK by ALTER at all, so it states them in the
-- fresh DDL and reports the rest through `blunderdb verify`), and it means this
-- migration cannot fail on a database whose history predates the rule.
DO $$
DECLARE
    c record;
BEGIN
    FOR c IN SELECT * FROM (VALUES
        ('position',        'position_dice_1_range',           'dice_1 BETWEEN 0 AND 6'),
        ('position',        'position_dice_2_range',           'dice_2 BETWEEN 0 AND 6'),
        ('position',        'position_cube_value_sign',        'cube_value >= 0'),
        ('position',        'position_pip_1_sign',             'pip_1 >= 0'),
        ('position',        'position_pip_2_sign',             'pip_2 >= 0'),
        ('position',        'position_off_1_range',            'off_1 BETWEEN 0 AND 15'),
        ('position',        'position_off_2_range',            'off_2 BETWEEN 0 AND 15'),
        ('anki_review_log', 'anki_review_log_rating_range',    'rating BETWEEN 1 AND 4')
    ) AS t(tbl, name, expr) LOOP
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint pc
            JOIN pg_class rel ON rel.oid = pc.conrelid
            JOIN pg_namespace n ON n.oid = rel.relnamespace
            WHERE n.nspname = current_schema() AND rel.relname = c.tbl AND pc.conname = c.name
        ) THEN
            EXECUTE format('ALTER TABLE %I ADD CONSTRAINT %I CHECK (%s) NOT VALID',
                           c.tbl, c.name, c.expr);
        END IF;
    END LOOP;
END $$;

UPDATE metadata SET value = '2.18.0' WHERE key = 'database_version';
