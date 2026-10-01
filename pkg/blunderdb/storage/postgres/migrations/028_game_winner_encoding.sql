-- Forward migration: the 2.26.0 wave — one encoding for game.winner
-- (1 = player 1, -1 = player 2, 0 = unfinished; domain.WinnerPlayer1).
--
-- The UPDATE below is sqlshared.NormalizeGameWinnerSQL verbatim, the statement
-- SQLite's migrate_2_25_0_to_2_26_0 runs; its doc gives the rule per source.
-- TestWinnerMigrationSQLIsShared keeps the two identical.
--
-- Row-Level Security, once applied, is FORCEd on game, match and import_batch:
-- the migrating connection carries no tenant and would see no row, so the
-- UPDATE would convert nothing and say nothing. FORCE is lifted for the
-- statement and put back; the batch runs as one transaction.

CREATE TEMP TABLE winner_migration_forced (name TEXT) ON COMMIT DROP;

DO $$
DECLARE t TEXT;
BEGIN
    FOREACH t IN ARRAY ARRAY['game', 'match', 'import_batch'] LOOP
        IF EXISTS (SELECT 1 FROM pg_class WHERE oid = to_regclass(t) AND relforcerowsecurity) THEN
            EXECUTE format('ALTER TABLE %I NO FORCE ROW LEVEL SECURITY', t);
            INSERT INTO winner_migration_forced VALUES (t);
        END IF;
    END LOOP;
END $$;

-- BEGIN NormalizeGameWinnerSQL
WITH g AS (
    SELECT g.id, g.match_id,
           COALESCE(g.winner, 0) AS w,
           COALESCE(g.points_won, 0) AS p,
           COALESCE(g.initial_score_1, 0) AS s1,
           COALESCE(g.initial_score_2, 0) AS s2,
           LEAD(COALESCE(g.initial_score_1, 0)) OVER (PARTITION BY g.match_id ORDER BY g.game_number, g.id) AS n1,
           LEAD(COALESCE(g.initial_score_2, 0)) OVER (PARTITION BY g.match_id ORDER BY g.game_number, g.id) AS n2,
           COALESCE(m.match_length, 0) AS len,
           CASE WHEN LOWER(COALESCE(b.format, '')) = 'bgf' OR LOWER(COALESCE(m.file_path, '')) LIKE '%.bgf' THEN 'bgf'
                WHEN LOWER(COALESCE(b.format, '')) = 'xg' OR LOWER(COALESCE(m.file_path, '')) LIKE '%.xg' THEN 'xg'
                ELSE 'gnubg' END AS hint
      FROM game g
      JOIN match m ON m.id = g.match_id
      LEFT JOIN import_batch b ON b.id = m.import_batch_id
), t AS (
    SELECT g.*,
           CASE WHEN p <= 0 THEN 0
                WHEN n1 IS NOT NULL AND n1 > s1 AND n2 = s2 THEN 1
                WHEN n1 IS NOT NULL AND n2 > s2 AND n1 = s1 THEN -1
           END AS truth
      FROM g
), v AS (
    SELECT match_id,
           SUM(CASE WHEN (truth = 1 AND w = 1) OR (truth = -1 AND w = -1) THEN 1 ELSE 0 END) AS xg,
           SUM(CASE WHEN (truth = 1 AND w = 0) OR (truth = -1 AND w = 1) THEN 1 ELSE 0 END) AS gnubg,
           SUM(CASE WHEN (truth = 1 AND w = -1) OR (truth = -1 AND w = 0) THEN 1 ELSE 0 END) AS swapped
      FROM t
     GROUP BY match_id
), e AS (
    SELECT t.*,
           CASE WHEN t.hint = 'bgf' THEN ''
                WHEN v.xg > 0 AND v.gnubg = 0 AND v.swapped = 0 THEN 'xg'
                WHEN v.gnubg > 0 AND v.xg = 0 AND v.swapped = 0 THEN 'gnubg'
                WHEN v.swapped > 0 AND v.xg = 0 AND v.gnubg = 0 THEN 'swapped'
                ELSE '' END AS voted
      FROM t
      JOIN v ON v.match_id = t.match_id
), c AS (
    SELECT id,
           CASE WHEN truth IS NOT NULL THEN truth
                WHEN voted = 'xg' AND w IN (1, -1) THEN w
                WHEN voted = 'gnubg' AND w = 0 THEN 1
                WHEN voted = 'gnubg' AND w = 1 THEN -1
                WHEN voted = 'swapped' AND w = -1 THEN 1
                WHEN voted = 'swapped' AND w = 0 THEN -1
                WHEN voted <> '' THEN 0
                WHEN n1 IS NULL AND len > 0 AND s1 + p >= len AND s2 + p < len THEN 1
                WHEN n1 IS NULL AND len > 0 AND s2 + p >= len AND s1 + p < len THEN -1
                WHEN hint = 'xg' AND w IN (1, -1) THEN w
                WHEN hint = 'gnubg' AND w = 0 THEN 1
                WHEN hint = 'gnubg' AND w = 1 THEN -1
                ELSE 0 END AS nw
      FROM e
)
UPDATE game SET winner = c.nw
  FROM c
 WHERE game.id = c.id;
-- END NormalizeGameWinnerSQL

DO $$
DECLARE t TEXT;
BEGIN
    FOR t IN SELECT name FROM winner_migration_forced LOOP
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
    END LOOP;
END $$;
