-- Forward migration: the 2.52.0 wave — the change journal.
--
--   * change_log: one row per object (tenant_id, uuid), moved to the end by a
--     new seq each time the object is created, updated, deleted or trashed.
--     A reader past a cursor sees each object once, in its final state.
--   * blunderdb_journal(): the trigger function. It runs as the invoking role
--     (no SECURITY DEFINER), so the tenant policy below governs what it writes.
--   * One row trigger per uuid-bearing table. A uuid that goes from NULL to a
--     value (the backfill of 054) is not a change; a uuid that changes (a
--     restore from the trash giving back the identifier) drops the entry of
--     the fresh draw it replaces and creates the new one.
-- No COMMIT, no role switch: it holds on blunderdb_user under FORCE RLS.
--   * An analysis has no uuid of its own: a trigger on it journals its
--     position (an update), so a re-analysis reaches a peer.
-- CREATE TRIGGER takes SHARE ROW EXCLUSIVE on its table (reads pass, writes
-- wait), held on every table below until this file's transaction ends:
-- lock_timeout makes a busy table fail the migration, retried at the next
-- start, rather than queue writers behind a long transaction indefinitely.
-- Idempotent. Schema-visible: bumps domain.DatabaseVersion to 2.52.0.

SET LOCAL lock_timeout = '15s';

CREATE SEQUENCE IF NOT EXISTS change_log_seq;

CREATE TABLE IF NOT EXISTS change_log (
    tenant_id  BIGINT      NOT NULL,
    seq        BIGINT      NOT NULL DEFAULT nextval('change_log_seq'),
    uuid       UUID        NOT NULL,
    family     TEXT        NOT NULL,
    op         TEXT        NOT NULL CHECK (op IN ('create', 'update', 'delete', 'trash')),
    changed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- The writer's transaction: a reader shows an entry only once every
    -- transaction older than it has ended (pg_snapshot_xmin), so a cursor never
    -- passes the seq of a writer still in flight.
    xid        xid8        NOT NULL DEFAULT pg_current_xact_id(),
    PRIMARY KEY (tenant_id, uuid)
);
CREATE INDEX IF NOT EXISTS idx_change_log_seq ON change_log (tenant_id, seq);
CREATE INDEX IF NOT EXISTS idx_change_log_xid ON change_log (tenant_id, xid, seq);

DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = current_schema() AND c.relname = 'position' AND c.relrowsecurity
    ) THEN
        ALTER TABLE change_log ENABLE ROW LEVEL SECURITY;
        ALTER TABLE change_log FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS tenant_isolation ON change_log;
        CREATE POLICY tenant_isolation ON change_log
            USING      (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint);
    END IF;
END $$;

CREATE OR REPLACE FUNCTION blunderdb_journal() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    op text;
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF OLD.uuid IS NOT NULL THEN
            INSERT INTO change_log (tenant_id, uuid, family, op) VALUES (OLD.tenant_id, OLD.uuid, TG_ARGV[0], 'delete')
            ON CONFLICT (tenant_id, uuid) DO UPDATE
                SET seq = nextval('change_log_seq'), family = EXCLUDED.family, op = 'delete', changed_at = now(), xid = pg_current_xact_id();
        END IF;
        RETURN OLD;
    END IF;
    IF NEW.uuid IS NULL THEN
        RETURN NEW;
    END IF;
    op := 'create';
    IF TG_OP = 'UPDATE' THEN
        IF OLD.uuid IS NULL THEN
            RETURN NEW;
        ELSIF OLD.uuid IS DISTINCT FROM NEW.uuid THEN
            DELETE FROM change_log WHERE tenant_id = OLD.tenant_id AND uuid = OLD.uuid;
        ELSE
            op := 'update';
        END IF;
    END IF;
    INSERT INTO change_log (tenant_id, uuid, family, op) VALUES (NEW.tenant_id, NEW.uuid, TG_ARGV[0], op)
    ON CONFLICT (tenant_id, uuid) DO UPDATE
        SET seq = nextval('change_log_seq'), family = EXCLUDED.family, op = EXCLUDED.op, changed_at = now(), xid = pg_current_xact_id();
    RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS change_log_position ON position;
CREATE TRIGGER change_log_position AFTER INSERT OR UPDATE OR DELETE ON position
    FOR EACH ROW EXECUTE FUNCTION blunderdb_journal('position');
DROP TRIGGER IF EXISTS change_log_comment ON comment;
CREATE TRIGGER change_log_comment AFTER INSERT OR UPDATE OR DELETE ON comment
    FOR EACH ROW EXECUTE FUNCTION blunderdb_journal('comment');
DROP TRIGGER IF EXISTS change_log_collection ON collection;
CREATE TRIGGER change_log_collection AFTER INSERT OR UPDATE OR DELETE ON collection
    FOR EACH ROW EXECUTE FUNCTION blunderdb_journal('collection');
DROP TRIGGER IF EXISTS change_log_match ON match;
CREATE TRIGGER change_log_match AFTER INSERT OR DELETE ON match
    FOR EACH ROW EXECUTE FUNCTION blunderdb_journal('match');
-- The columns that belong to the library (where the user last looked, the
-- tournament, the import batch) stay out of what a peer receives, so changing
-- them alone dates nothing. sqlshared.LocalMatchColumns spells the same list.
DROP TRIGGER IF EXISTS change_log_match_update ON match;
CREATE TRIGGER change_log_match_update AFTER UPDATE ON match
    FOR EACH ROW WHEN (
        (to_jsonb(OLD) - ARRAY['import_date', 'tournament_id', 'import_batch_id', 'last_visited_position', 'last_visited_at'])
        IS DISTINCT FROM
        (to_jsonb(NEW) - ARRAY['import_date', 'tournament_id', 'import_batch_id', 'last_visited_position', 'last_visited_at'])
    ) EXECUTE FUNCTION blunderdb_journal('match');
DROP TRIGGER IF EXISTS change_log_transcription ON transcription;
CREATE TRIGGER change_log_transcription AFTER INSERT OR UPDATE OR DELETE ON transcription
    FOR EACH ROW EXECUTE FUNCTION blunderdb_journal('transcription');
DROP TRIGGER IF EXISTS change_log_lesson ON lesson;
CREATE TRIGGER change_log_lesson AFTER INSERT OR UPDATE OR DELETE ON lesson
    FOR EACH ROW EXECUTE FUNCTION blunderdb_journal('lesson');
DROP TRIGGER IF EXISTS change_log_anki_card ON anki_card;
CREATE TRIGGER change_log_anki_card AFTER INSERT OR UPDATE OR DELETE ON anki_card
    FOR EACH ROW EXECUTE FUNCTION blunderdb_journal('anki_card');

CREATE OR REPLACE FUNCTION blunderdb_journal_analysis() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    u uuid;
BEGIN
    SELECT p.uuid INTO u FROM position p WHERE p.id = NEW.position_id AND p.tenant_id = NEW.tenant_id;
    IF u IS NOT NULL THEN
        INSERT INTO change_log (tenant_id, uuid, family, op) VALUES (NEW.tenant_id, u, 'position', 'update')
        ON CONFLICT (tenant_id, uuid) DO UPDATE
            SET seq = nextval('change_log_seq'), family = 'position', op = 'update', changed_at = now(), xid = pg_current_xact_id();
    END IF;
    RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS change_log_analysis ON analysis;
CREATE TRIGGER change_log_analysis AFTER INSERT OR UPDATE ON analysis
    FOR EACH ROW EXECUTE FUNCTION blunderdb_journal_analysis();
