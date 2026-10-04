-- Forward migration: the 2.31.0 wave.
--
--   * match_equity_table — the match equity tables a tenant imported from
--     gnubg .xml files, identified by the digest of their values; at most one
--     is current, none current means the built-in Kazaross-XG2 (ADR-0068).
--   * analysis.met_digest — the table an analysis was computed with, NULL for
--     the built-in one. An analysis made with another table than the current
--     one is marked "different MET" and left out of the comparisons.
--   * lesson_progress — the Steps of a Lesson the student marked done, by
--     the explicit gesture only (ADR-0069). No exporter reads it.
--   * move.error_mp — the equity a play gave up, in millipoints. Added NULL
--     and left NULL here: ADD COLUMN without a default rewrites nothing, so
--     the migration stays instant on a 15 M-position library. Existing moves
--     are scored by the resumable pass MatchStore.ScoreMoves, outside the
--     migration.
-- Schema-visible: bumps domain.DatabaseVersion to 2.31.0.

ALTER TABLE analysis ADD COLUMN IF NOT EXISTS met_digest TEXT;
ALTER TABLE move     ADD COLUMN IF NOT EXISTS error_mp   INTEGER;

CREATE TABLE IF NOT EXISTS match_equity_table (
    id          BIGSERIAL   PRIMARY KEY,
    tenant_id   BIGINT      NOT NULL,
    name        TEXT        NOT NULL,
    digest      TEXT        NOT NULL,
    source      TEXT        NOT NULL,
    is_current  INTEGER     NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ DEFAULT now(),
    CONSTRAINT match_equity_table_tenant_digest_key UNIQUE (tenant_id, digest)
);

-- A Step's progress row must belong to the Step's own tenant: the composite
-- key needs (tenant_id, id) to be unique on lesson_step.
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'lesson_step_tenant_id_key') THEN
        ALTER TABLE lesson_step ADD CONSTRAINT lesson_step_tenant_id_key UNIQUE (tenant_id, id);
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS lesson_progress (
    lesson_step_id BIGINT      PRIMARY KEY,
    tenant_id      BIGINT      NOT NULL,
    done_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT lesson_progress_step_tenant_fkey
        FOREIGN KEY (tenant_id, lesson_step_id) REFERENCES lesson_step (tenant_id, id) ON DELETE CASCADE
);

-- Row-level security, applied only if this schema already has it. Same shape
-- as 031_lesson.sql.
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = current_schema() AND c.relname = 'position' AND c.relrowsecurity
    ) THEN
        ALTER TABLE match_equity_table ENABLE ROW LEVEL SECURITY;
        ALTER TABLE match_equity_table FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS tenant_isolation ON match_equity_table;
        CREATE POLICY tenant_isolation ON match_equity_table
            USING      (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint);
        ALTER TABLE lesson_progress ENABLE ROW LEVEL SECURITY;
        ALTER TABLE lesson_progress FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS tenant_isolation ON lesson_progress;
        CREATE POLICY tenant_isolation ON lesson_progress
            USING      (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint);
    END IF;
END $$;
