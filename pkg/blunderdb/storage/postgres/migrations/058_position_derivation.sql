-- Forward migration: the derivation of a position — the position it derives
-- from and the move played to reach it (a variant's link to its origin).
--
-- A new table: no existing row is rewritten. A derivation goes with either of its two
-- positions (composite tenant keys, ON DELETE CASCADE: deleting the origin
-- drops the link, never the variant) and travels in exports, transfers and the
-- change journal as part of the variant: a trigger journals it (an update)
-- whenever its link is written (function defined in 057). A position has one
-- origin.
-- No COMMIT, no role switch: it holds on blunderdb_user under FORCE RLS.
-- CREATE TABLE and the foreign keys take SHARE ROW EXCLUSIVE on "position"
-- (reads pass, writes wait) until this file's transaction ends: lock_timeout
-- makes a busy table fail the migration, retried at the next start, rather
-- than queue writers behind a long transaction. Idempotent.
-- Schema-visible: bumps domain.DatabaseVersion to 2.55.0.

SET LOCAL lock_timeout = '15s';

CREATE TABLE IF NOT EXISTS position_derivation (
    id               BIGSERIAL PRIMARY KEY,
    tenant_id        BIGINT  NOT NULL,
    position_id      BIGINT  NOT NULL,
    from_position_id BIGINT  NOT NULL,
    move             TEXT    NOT NULL DEFAULT '' CHECK (length(move) <= 100),
    CONSTRAINT position_derivation_position_key UNIQUE (tenant_id, position_id),
    CONSTRAINT position_derivation_not_self CHECK (position_id <> from_position_id),
    -- Composite keys: both ends are positions of the row's own tenant
    -- (017_composite_tenant_fk.sql).
    CONSTRAINT position_derivation_position_tenant_fkey
        FOREIGN KEY (tenant_id, position_id) REFERENCES position (tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT position_derivation_from_tenant_fkey
        FOREIGN KEY (tenant_id, from_position_id) REFERENCES position (tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_position_derivation_from ON position_derivation (tenant_id, from_position_id);

-- Row-level security, applied only if this schema already has it. Same shape
-- as 057_position_drawing.sql.
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = current_schema() AND c.relname = 'position' AND c.relrowsecurity
    ) THEN
        ALTER TABLE position_derivation ENABLE ROW LEVEL SECURITY;
        ALTER TABLE position_derivation FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS tenant_isolation ON position_derivation;
        CREATE POLICY tenant_isolation ON position_derivation
            USING      (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint);
    END IF;
END $$;

DROP TRIGGER IF EXISTS change_log_position_derivation ON position_derivation;
CREATE TRIGGER change_log_position_derivation AFTER INSERT OR UPDATE OR DELETE ON position_derivation
    FOR EACH ROW EXECUTE FUNCTION blunderdb_journal_of_position();
