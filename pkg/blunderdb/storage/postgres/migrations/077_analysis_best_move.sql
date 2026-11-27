-- Forward migration: analysis.best_move, the best checker play of the stored
-- analysis lower-cased (engine.BestMove), which a move-pattern search narrows
-- by before decoding any blob.
--
--   * NULL means "not derived yet": every row this migration finds. The search
--     lets such a row through and settles it on the blob, so the column is
--     correct before the Go backfill (runGoBackfills) has filled it.
--   * '' is an analysis with no checker play (a cube analysis).
--
-- idx_analysis_best_move: LIKE is not leakproof, so under row-level security
-- no index, a pg_trgm one included, can answer '%pattern%' as an index
-- condition. The tenant's best moves are read from this index alone instead,
-- short strings, without the heap and its blobs.
--
-- blunderdb_best_move_stale: a binary older than 2.61 rewrites data and leaves
-- best_move as it was, which may name the former best play and hide the row
-- from a search. Any UPDATE OF data that keeps best_move sets it NULL instead;
-- a binary that knows the column writes it again in a second statement
-- (analysisBestMoveSQL) when the play is unchanged. The trigger only edits NEW:
-- it reads no table, so FORCEd RLS does not bind it, and it writes no journal.
--
-- Derived from the blob, which travels: no journal entry, no reader floor.
-- Idempotent. CREATE INDEX holds a SHARE lock on analysis until this file's
-- transaction ends; lock_timeout makes a busy table fail the migration,
-- retried at the next start.
-- Schema-visible: bumps domain.DatabaseVersion to 2.61.0.

SET LOCAL lock_timeout = '15s';

ALTER TABLE analysis ADD COLUMN IF NOT EXISTS best_move TEXT;

CREATE INDEX IF NOT EXISTS idx_analysis_best_move ON analysis (tenant_id, best_move) INCLUDE (position_id);

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
