-- Forward migration: the 2.29.0 wave — the Lesson (ADR-0066).
--
-- A Lesson is an ordered sequence of Steps a coach writes; each Step is a
-- text that may show a Collection, a Position, both or neither. A deleted
-- Collection or Position leaves the Step and its text (SET NULL names its
-- column, 026_set_null_names_its_column.sql); deleting the Lesson takes its
-- Steps. A Position a Step shows is held by the retention predicate.
-- Schema-visible: bumps domain.DatabaseVersion to 2.29.0.

CREATE TABLE IF NOT EXISTS lesson (
    id          BIGSERIAL   PRIMARY KEY,
    tenant_id   BIGINT      NOT NULL,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ DEFAULT now(),
    updated_at  TIMESTAMPTZ DEFAULT now(),
    CONSTRAINT lesson_tenant_id_key UNIQUE (tenant_id, id)
);

CREATE TABLE IF NOT EXISTS lesson_step (
    id            BIGSERIAL PRIMARY KEY,
    tenant_id     BIGINT    NOT NULL,
    lesson_id     BIGINT    NOT NULL,
    sort_order    INTEGER   NOT NULL DEFAULT 0,
    title         TEXT      NOT NULL DEFAULT '',
    text          TEXT      NOT NULL DEFAULT '',
    collection_id BIGINT,
    position_id   BIGINT,
    -- Composite keys: a Step can only belong to, and point at, rows of its
    -- own tenant (017_composite_tenant_fk.sql).
    CONSTRAINT lesson_step_lesson_tenant_fkey
        FOREIGN KEY (tenant_id, lesson_id) REFERENCES lesson (tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT lesson_step_collection_tenant_fkey
        FOREIGN KEY (tenant_id, collection_id) REFERENCES collection (tenant_id, id) ON DELETE SET NULL (collection_id),
    CONSTRAINT lesson_step_position_tenant_fkey
        FOREIGN KEY (tenant_id, position_id) REFERENCES position (tenant_id, id) ON DELETE SET NULL (position_id)
);

CREATE INDEX IF NOT EXISTS idx_lesson_step_lesson ON lesson_step(lesson_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_lesson_step_position ON lesson_step(position_id);

-- Row-level security, applied only if this schema already has it. Same shape
-- as 027_rencontre.sql.
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = current_schema() AND c.relname = 'position' AND c.relrowsecurity
    ) THEN
        ALTER TABLE lesson ENABLE ROW LEVEL SECURITY;
        ALTER TABLE lesson FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS tenant_isolation ON lesson;
        CREATE POLICY tenant_isolation ON lesson
            USING      (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint);
        ALTER TABLE lesson_step ENABLE ROW LEVEL SECURITY;
        ALTER TABLE lesson_step FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS tenant_isolation ON lesson_step;
        CREATE POLICY tenant_isolation ON lesson_step
            USING      (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint);
    END IF;
END $$;
