-- Forward migration: the study mark of the library-wide study backlog.
--
-- One row per position the user marked "studied": written only by that
-- gesture, reversible, never exported. Not a retention reason. Part of the
-- 2.31.0 schema; this file does not write database_version.

CREATE TABLE IF NOT EXISTS study_mark (
    position_id BIGINT NOT NULL,
    tenant_id   BIGINT NOT NULL,
    marked_at   BIGINT NOT NULL,
    PRIMARY KEY (tenant_id, position_id),
    -- Composite key: a mark can only point at a position of its own tenant
    -- (017_composite_tenant_fk.sql).
    CONSTRAINT study_mark_position_tenant_fkey
        FOREIGN KEY (tenant_id, position_id) REFERENCES position (tenant_id, id) ON DELETE CASCADE
);

-- Row-level security, applied only if this schema already has it. Same shape
-- as 031_lesson.sql.
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = current_schema() AND c.relname = 'position' AND c.relrowsecurity
    ) THEN
        ALTER TABLE study_mark ENABLE ROW LEVEL SECURITY;
        ALTER TABLE study_mark FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS tenant_isolation ON study_mark;
        CREATE POLICY tenant_isolation ON study_mark
            USING      (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint);
    END IF;
END $$;
