-- Forward migration: a match stores its result (domain.Match.FinalScore and
-- Winner), computed at import from its games, so a list of matches shows it
-- without reading them.
--
--   * match.final_score_1, final_score_2 — the score after the last game,
--     NULL for a match without games.
--   * match.winner — 1 or -1 for the seat that reached the length, 0 for none.
--
-- The result is derived from the games, which travel: it is local to the
-- library like the attachment columns, so computing it here dates no match.
-- The journal trigger learns it before the backfill runs.
-- Idempotent: the backfill reads only the rows still without a result.
-- Schema-visible: bumps domain.DatabaseVersion to 2.58.0.

ALTER TABLE match ADD COLUMN IF NOT EXISTS final_score_1 INTEGER;
ALTER TABLE match ADD COLUMN IF NOT EXISTS final_score_2 INTEGER;
ALTER TABLE match ADD COLUMN IF NOT EXISTS winner INTEGER NOT NULL DEFAULT 0;

DROP TRIGGER IF EXISTS change_log_match_update ON match;
CREATE TRIGGER change_log_match_update AFTER UPDATE ON match
    FOR EACH ROW WHEN (
        (to_jsonb(OLD) - ARRAY['import_date', 'tournament_id', 'tournament_sort_order', 'import_batch_id', 'last_visited_position', 'last_visited_at', 'direction_match_id', 'final_score_1', 'final_score_2', 'winner'])
        IS DISTINCT FROM
        (to_jsonb(NEW) - ARRAY['import_date', 'tournament_id', 'tournament_sort_order', 'import_batch_id', 'last_visited_position', 'last_visited_at', 'direction_match_id', 'final_score_1', 'final_score_2', 'winner'])
    ) EXECUTE FUNCTION blunderdb_journal('match');

-- Row-Level Security, once applied, is FORCEd on the tables the backfill
-- reads and writes: the migrating connection carries no tenant and would see
-- no row. FORCE is lifted for the statements and put back.
CREATE TEMP TABLE match_result_forced (name TEXT) ON COMMIT DROP;

DO $$
DECLARE t TEXT;
BEGIN
    FOREACH t IN ARRAY ARRAY['game', 'match'] LOOP
        IF EXISTS (SELECT 1 FROM pg_class WHERE oid = to_regclass(t) AND relforcerowsecurity) THEN
            EXECUTE format('ALTER TABLE %I NO FORCE ROW LEVEL SECURITY', t);
            INSERT INTO match_result_forced VALUES (t);
        END IF;
    END LOOP;
END $$;

UPDATE match m SET final_score_1 = r.s1, final_score_2 = r.s2,
       winner = CASE
           WHEN COALESCE(m.match_length, 0) > 0 AND r.s1 >= m.match_length THEN 1
           WHEN COALESCE(m.match_length, 0) > 0 AND r.s2 >= m.match_length THEN -1
           ELSE 0 END
  FROM (SELECT DISTINCT ON (g.tenant_id, g.match_id) g.tenant_id, g.match_id,
               COALESCE(g.initial_score_1, 0) + CASE WHEN g.winner = 1 THEN COALESCE(g.points_won, 0) ELSE 0 END AS s1,
               COALESCE(g.initial_score_2, 0) + CASE WHEN g.winner = -1 THEN COALESCE(g.points_won, 0) ELSE 0 END AS s2
          FROM game g
         ORDER BY g.tenant_id, g.match_id, g.game_number DESC, g.id DESC) r
 WHERE m.final_score_1 IS NULL AND m.tenant_id = r.tenant_id AND m.id = r.match_id;

DO $$
DECLARE t TEXT;
BEGIN
    FOR t IN SELECT name FROM match_result_forced LOOP
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
    END LOOP;
END $$;
