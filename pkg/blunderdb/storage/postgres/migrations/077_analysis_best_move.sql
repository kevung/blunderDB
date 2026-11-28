-- Forward migration: analysis.best_move, the best checker play of the stored
-- analysis lower-cased (engine.BestMove), which a move-pattern search narrows
-- by before decoding any blob.
--
--   * NULL means "not derived yet": every row this migration finds. The search
--     lets such a row through and settles it on the blob, so the column is
--     correct before the Go backfill (runBestMoveBackfill) has filled it.
--   * '' is an analysis with no checker play (a cube analysis).
--
-- Only what is instant holds the ACCESS EXCLUSIVE lock ADD COLUMN takes: a
-- nullable column without default touches no row. Its index is built by
-- ensureBestMoveIndex, CONCURRENTLY, outside any transaction.
--
-- metadata.best_move_backfill_pending lists the tenants holding analyses,
-- read under the lock already held (FORCE lifted for the listing, a skip
-- scan of idx_analysis_tenant_position_key): the backfill then reads each
-- tenant bound to it, without lifting FORCE again.
--
-- blunderdb_best_move_stale: a binary older than 2.61 rewrites data and leaves
-- best_move as it was, which may name the former best play and hide the row
-- from a search. Any UPDATE OF data that keeps best_move sets it NULL instead;
-- a binary that knows the column writes it again in a second statement
-- (analysisBestMoveSQL) when the play is unchanged. The trigger only edits NEW:
-- it reads no table, so FORCEd RLS does not bind it, and it writes no journal.
--
-- Derived from the blob, which travels: no journal entry, no reader floor.
-- Idempotent. lock_timeout makes a busy table fail the migration, retried at
-- the next start.
-- Schema-visible: bumps domain.DatabaseVersion to 2.61.0.

SET LOCAL lock_timeout = '15s';

ALTER TABLE analysis ADD COLUMN IF NOT EXISTS best_move TEXT;

DO $$
DECLARE
    forced boolean;
    tenants text;
BEGIN
    SELECT relforcerowsecurity INTO forced FROM pg_class WHERE oid = 'analysis'::regclass;
    IF forced THEN
        ALTER TABLE analysis NO FORCE ROW LEVEL SECURITY;
    END IF;
    WITH RECURSIVE t(id) AS (
        (SELECT tenant_id FROM analysis ORDER BY tenant_id LIMIT 1)
        UNION ALL
        SELECT (SELECT a.tenant_id FROM analysis a WHERE a.tenant_id > t.id ORDER BY a.tenant_id LIMIT 1)
          FROM t WHERE t.id IS NOT NULL
    )
    SELECT string_agg(id::text, ' ' ORDER BY id) INTO tenants FROM t WHERE id IS NOT NULL;
    IF forced THEN
        ALTER TABLE analysis FORCE ROW LEVEL SECURITY;
    END IF;
    IF tenants IS NOT NULL THEN
        INSERT INTO metadata (key, value) VALUES ('best_move_backfill_pending', '0 ' || tenants)
        ON CONFLICT (key) DO NOTHING;
    END IF;
END $$;

CREATE OR REPLACE FUNCTION blunderdb_best_move_stale() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    NEW.best_move := NULL;
    RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS analysis_best_move_stale ON analysis;
CREATE TRIGGER analysis_best_move_stale BEFORE UPDATE OF data ON analysis
    FOR EACH ROW WHEN (NEW.best_move IS NOT DISTINCT FROM OLD.best_move)
    EXECUTE FUNCTION blunderdb_best_move_stale();
