-- Forward migration: a journal that writes less (no schema change a reader
-- sees, DatabaseVersion untouched). Functions only: nothing is rewritten and no
-- lock is held. Back: the three functions of 059.
--
-- A second write of an object in the transaction that already journaled it adds
-- nothing a reader can see: nobody reads the entry before the commit, and the
-- object travels in its final state. Skipping it spares a dead row per position
-- on import (406 to 206 bytes a position). What still moves the entry in the same
-- transaction: a delete or a trash undone, a uuid given anew (op create), and an
-- entry dated elsewhere (Mark with a date, older than this transaction's now).
--
-- A library may keep no journal at all (a public library no peer syncs with:
-- storage.ChangeJournalStore.SetJournalOff). The setting is read under the
-- tenant's policy, so another tenant's choice is out of reach. An unset tenant
-- reads "on".
-- No COMMIT, no role switch: it holds on blunderdb_user under FORCE RLS.

CREATE OR REPLACE FUNCTION blunderdb_journal_off(t bigint) RETURNS boolean
LANGUAGE sql STABLE AS $$
    SELECT EXISTS (SELECT 1 FROM library_settings WHERE tenant_id = t AND key = 'change_journal' AND value = 'off')
$$;

CREATE OR REPLACE FUNCTION blunderdb_journal_same_write(entry_xid xid8, entry_op text, entry_at timestamptz, new_op text) RETURNS boolean
LANGUAGE sql STABLE AS $$
    SELECT entry_xid = pg_current_xact_id() AND new_op = 'update' AND entry_op IN ('create', 'update') AND entry_at >= now()
$$;

CREATE OR REPLACE FUNCTION blunderdb_journal() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    op text;
    applying timestamptz := blunderdb_applying_at();
    stamp timestamptz := now();
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF blunderdb_journal_off(OLD.tenant_id) THEN
            RETURN OLD;
        END IF;
        IF OLD.uuid IS NOT NULL THEN
            INSERT INTO change_log (tenant_id, uuid, family, op, changed_at) VALUES (OLD.tenant_id, OLD.uuid, TG_ARGV[0], 'delete', COALESCE(applying, stamp))
            ON CONFLICT (tenant_id, uuid) DO UPDATE
                SET seq = nextval('change_log_seq'), family = EXCLUDED.family, op = 'delete',
                    changed_at = CASE WHEN applying IS NULL THEN blunderdb_stamp_after(change_log.changed_at) ELSE EXCLUDED.changed_at END, xid = pg_current_xact_id();
        END IF;
        RETURN OLD;
    END IF;
    IF NEW.uuid IS NULL OR blunderdb_journal_off(NEW.tenant_id) THEN
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
            changed_at = CASE WHEN applying IS NULL THEN blunderdb_stamp_after(change_log.changed_at) ELSE EXCLUDED.changed_at END, xid = pg_current_xact_id()
        WHERE NOT blunderdb_journal_same_write(change_log.xid, change_log.op, change_log.changed_at, EXCLUDED.op);
    RETURN NEW;
END $$;

CREATE OR REPLACE FUNCTION blunderdb_journal_analysis() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    u uuid;
BEGIN
    IF blunderdb_applying_at() IS NOT NULL OR blunderdb_journal_off(NEW.tenant_id) THEN
        RETURN NEW;
    END IF;
    SELECT p.uuid INTO u FROM position p WHERE p.id = NEW.position_id AND p.tenant_id = NEW.tenant_id;
    IF u IS NOT NULL THEN
        INSERT INTO change_log (tenant_id, uuid, family, op) VALUES (NEW.tenant_id, u, 'position', 'update')
        ON CONFLICT (tenant_id, uuid) DO UPDATE
            SET seq = nextval('change_log_seq'), family = 'position', op = 'update', changed_at = blunderdb_stamp_after(change_log.changed_at), xid = pg_current_xact_id()
          WHERE NOT blunderdb_journal_same_write(change_log.xid, change_log.op, change_log.changed_at, EXCLUDED.op);
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
    IF u IS NOT NULL AND NOT blunderdb_journal_off(tid) THEN
        INSERT INTO change_log (tenant_id, uuid, family, op) VALUES (tid, u, 'position', 'update')
        ON CONFLICT (tenant_id, uuid) DO UPDATE
            SET seq = nextval('change_log_seq'), family = 'position', op = 'update', changed_at = blunderdb_stamp_after(change_log.changed_at), xid = pg_current_xact_id()
        WHERE NOT blunderdb_journal_same_write(change_log.xid, change_log.op, change_log.changed_at, EXCLUDED.op);
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
