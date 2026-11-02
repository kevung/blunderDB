-- Forward migration: the drawing of a position — the arrows and squares a
-- player lays over its board, in the order drawn.
--
-- A new table: no existing row is rewritten. A drawing goes with its position
-- (composite tenant key, ON DELETE CASCADE) and travels in exports, transfers
-- and the change journal as part of it: a trigger journals the position (an
-- update) whenever a mark is written, as 055 does for an analysis.
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

-- When a drawing or a link (058) was last written, an erasure included, and the
-- link whose origin has not arrived yet: what tells "erased" from "never heard
-- of". Written by the trigger below and by an import; it travels in the
-- position's record. SQLite's twin is in its schema.
CREATE TABLE IF NOT EXISTS position_part_fact (
    tenant_id      BIGINT NOT NULL,
    position_id    BIGINT NOT NULL,
    part           TEXT   NOT NULL CHECK (part IN ('drawing', 'derivation')),
    changed_at     TEXT   NOT NULL,
    pending_origin TEXT,
    pending_move   TEXT,
    PRIMARY KEY (tenant_id, position_id, part),
    CONSTRAINT position_part_fact_position_tenant_fkey
        FOREIGN KEY (tenant_id, position_id) REFERENCES position (tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_position_part_fact_pending ON position_part_fact (tenant_id, pending_origin) WHERE pending_origin IS NOT NULL;

DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = current_schema() AND c.relname = 'position' AND c.relrowsecurity
    ) THEN
        ALTER TABLE position_part_fact ENABLE ROW LEVEL SECURITY;
        ALTER TABLE position_part_fact FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS tenant_isolation ON position_part_fact;
        CREATE POLICY tenant_isolation ON position_part_fact
            USING      (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint);
    END IF;
END $$;

-- Writing a mark of a position (or a link of a variant) is an update of that
-- position, as a re-analysis is (055): they have no uuid of their own, and a
-- peer must receive them with the position. When the position itself is going
-- (the cascade), it is already gone here and its own delete entry stands.
CREATE OR REPLACE FUNCTION blunderdb_journal_of_position() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    pid bigint;
    tid bigint;
    u uuid;
BEGIN
    IF blunderdb_applying_at() IS NOT NULL THEN
        RETURN NULL;
    END IF;
    IF TG_OP = 'DELETE' THEN
        pid := OLD.position_id; tid := OLD.tenant_id;
    ELSE
        pid := NEW.position_id; tid := NEW.tenant_id;
    END IF;
    SELECT p.uuid INTO u FROM position p WHERE p.id = pid AND p.tenant_id = tid;
    IF u IS NOT NULL THEN
        INSERT INTO change_log (tenant_id, uuid, family, op) VALUES (tid, u, 'position', 'update')
        ON CONFLICT (tenant_id, uuid) DO UPDATE
            SET seq = nextval('change_log_seq'), family = 'position', op = 'update', changed_at = now(), xid = pg_current_xact_id();
    END IF;
    IF EXISTS (SELECT 1 FROM position p WHERE p.id = pid AND p.tenant_id = tid) THEN
        INSERT INTO position_part_fact (tenant_id, position_id, part, changed_at)
        VALUES (tid, pid, substr(TG_TABLE_NAME, length('position_') + 1),
                to_char(now() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"'))
        ON CONFLICT (tenant_id, position_id, part) DO UPDATE
            SET changed_at = EXCLUDED.changed_at, pending_origin = NULL, pending_move = NULL;
    END IF;
    RETURN NULL;
END $$;

DROP TRIGGER IF EXISTS change_log_position_drawing ON position_drawing;
CREATE TRIGGER change_log_position_drawing AFTER INSERT OR UPDATE OR DELETE ON position_drawing
    FOR EACH ROW EXECUTE FUNCTION blunderdb_journal_of_position();
