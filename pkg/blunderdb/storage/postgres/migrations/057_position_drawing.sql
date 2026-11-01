-- Forward migration: the drawing of a position — the arrows and squares a
-- player lays over its board, in the order drawn.
--
-- A new table with no trigger: the change journal (055) is not touched, and
-- no existing row is rewritten. A drawing goes with its position (composite
-- tenant key, ON DELETE CASCADE) and travels in exports and transfers, not in
-- the journal.
-- No COMMIT, no role switch: it holds on blunderdb_user under FORCE RLS.
-- CREATE TABLE and the foreign key take SHARE ROW EXCLUSIVE on "position"
-- (reads pass, writes wait) until this file's transaction ends: lock_timeout
-- makes a busy table fail the migration, retried at the next start, rather
-- than queue writers behind a long transaction. Idempotent.
-- Schema-visible: bumps domain.DatabaseVersion to 2.54.0.

SET LOCAL lock_timeout = '15s';

CREATE TABLE IF NOT EXISTS position_drawing (
    id          BIGSERIAL PRIMARY KEY,
    tenant_id   BIGINT    NOT NULL,
    position_id BIGINT    NOT NULL,
    rank        INTEGER   NOT NULL,
    kind        TEXT      NOT NULL CHECK (kind IN ('arrow', 'square')),
    from_point  INTEGER   NOT NULL CHECK (from_point BETWEEN 0 AND 25),
    to_point    INTEGER   NOT NULL CHECK (to_point BETWEEN 0 AND 25),
    color       TEXT      NOT NULL CHECK (color IN ('red', 'green', 'blue', 'yellow')),
    CONSTRAINT position_drawing_rank_key UNIQUE (tenant_id, position_id, rank),
    -- Composite key: a mark can only point at a position of its own tenant
    -- (017_composite_tenant_fk.sql).
    CONSTRAINT position_drawing_position_tenant_fkey
        FOREIGN KEY (tenant_id, position_id) REFERENCES position (tenant_id, id) ON DELETE CASCADE
);

-- Row-level security, applied only if this schema already has it. Same shape
-- as 035_study_mark.sql.
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = current_schema() AND c.relname = 'position' AND c.relrowsecurity
    ) THEN
        ALTER TABLE position_drawing ENABLE ROW LEVEL SECURITY;
        ALTER TABLE position_drawing FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS tenant_isolation ON position_drawing;
        CREATE POLICY tenant_isolation ON position_drawing
            USING      (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint);
    END IF;
END $$;
