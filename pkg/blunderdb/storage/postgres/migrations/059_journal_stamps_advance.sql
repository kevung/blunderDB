-- Forward migration: the 2.56.0 wave — dates of the journal that only grow.
--
-- Two writes of an object are ordered for a peer by their date alone. Dated by
-- the clock, the second can equal the first (the same millisecond) or precede it
-- (a clock set back); a peer holding the first then keeps it for good, and sends
-- it back over the later write. A local write is now dated now, or a millisecond
-- past what the object holds when now() has not passed it. Dates a peer's record
-- brings (blunderdb_applying_at, Mark with a date) are kept as they came.
--
-- Functions only, no table: nothing is rewritten and no lock is held. Idempotent.
-- No COMMIT, no role switch: it holds on blunderdb_user under FORCE RLS.
-- Schema-visible: bumps domain.DatabaseVersion to 2.56.0.

-- Dates compare at the millisecond once they leave the journal, whatever finer
-- part the column holds: the next date is counted from that millisecond.
CREATE OR REPLACE FUNCTION blunderdb_stamp_after(prev timestamptz) RETURNS timestamptz
LANGUAGE sql STABLE AS $$
    SELECT GREATEST(now(), date_trunc('milliseconds', prev) + interval '1 millisecond')
$$;

CREATE OR REPLACE FUNCTION blunderdb_journal() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    op text;
    applying timestamptz := blunderdb_applying_at();
    stamp timestamptz := now();
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF OLD.uuid IS NOT NULL THEN
            INSERT INTO change_log (tenant_id, uuid, family, op, changed_at) VALUES (OLD.tenant_id, OLD.uuid, TG_ARGV[0], 'delete', COALESCE(applying, stamp))
            ON CONFLICT (tenant_id, uuid) DO UPDATE
                SET seq = nextval('change_log_seq'), family = EXCLUDED.family, op = 'delete',
                    changed_at = CASE WHEN applying IS NULL THEN blunderdb_stamp_after(change_log.changed_at) ELSE EXCLUDED.changed_at END, xid = pg_current_xact_id();
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
    IF applying IS NOT NULL THEN
        IF op = 'update' THEN
            RETURN NEW;
        END IF;
        -- Below the floor: the row's real record always wins.
        stamp := '1969-12-31T00:00:00.000Z';
    END IF;
    INSERT INTO change_log (tenant_id, uuid, family, op, changed_at) VALUES (NEW.tenant_id, NEW.uuid, TG_ARGV[0], op, stamp)
    ON CONFLICT (tenant_id, uuid) DO UPDATE
        SET seq = nextval('change_log_seq'), family = EXCLUDED.family, op = EXCLUDED.op,
            changed_at = CASE WHEN applying IS NULL THEN blunderdb_stamp_after(change_log.changed_at) ELSE EXCLUDED.changed_at END, xid = pg_current_xact_id();
    RETURN NEW;
END $$;


CREATE OR REPLACE FUNCTION blunderdb_journal_analysis() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    u uuid;
BEGIN
    IF blunderdb_applying_at() IS NOT NULL THEN
        RETURN NEW;
    END IF;
    SELECT p.uuid INTO u FROM position p WHERE p.id = NEW.position_id AND p.tenant_id = NEW.tenant_id;
    IF u IS NOT NULL THEN
        INSERT INTO change_log (tenant_id, uuid, family, op) VALUES (NEW.tenant_id, u, 'position', 'update')
        ON CONFLICT (tenant_id, uuid) DO UPDATE
            SET seq = nextval('change_log_seq'), family = 'position', op = 'update', changed_at = blunderdb_stamp_after(change_log.changed_at), xid = pg_current_xact_id();
    END IF;
    RETURN NEW;
END $$;


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
            SET seq = nextval('change_log_seq'), family = 'position', op = 'update', changed_at = blunderdb_stamp_after(change_log.changed_at), xid = pg_current_xact_id();
    END IF;
    IF EXISTS (SELECT 1 FROM position p WHERE p.id = pid AND p.tenant_id = tid) THEN
        INSERT INTO position_part_fact (tenant_id, position_id, part, changed_at)
        VALUES (tid, pid, substr(TG_TABLE_NAME, length('position_') + 1),
                to_char(now() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"'))
        ON CONFLICT (tenant_id, position_id, part) DO UPDATE
            SET changed_at = to_char(blunderdb_stamp_after(position_part_fact.changed_at::timestamptz) AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"'),
                pending_origin = NULL, pending_move = NULL;
    END IF;
    RETURN NULL;
END $$;
