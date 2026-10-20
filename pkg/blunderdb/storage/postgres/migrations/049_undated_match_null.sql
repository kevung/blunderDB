-- Forward migration: the 2.45.0 wave — a match without a date of play is NULL.
--
--   * An edit that echoed an undated match's zero time back stored the year-1
--     day, which sorted as the oldest match instead of by import date. Every
--     such row goes to NULL, the one form of "no date".
--   * The positions those matches reach are re-dated: position.match_date was
--     derived from the year-1 day.
-- Idempotent: a second run finds no year-1 row.
-- Schema-visible: bumps domain.DatabaseVersion to 2.45.0.

-- Row-Level Security, once applied, is FORCEd on the tables these updates
-- read and write: the migrating connection carries no tenant and would see
-- no row, so the migration would change nothing and say nothing. FORCE is
-- lifted for the statements and put back; the batch runs as one transaction.
CREATE TEMP TABLE undated_match_forced (name TEXT) ON COMMIT DROP;

DO $$
DECLARE t TEXT;
BEGIN
    FOREACH t IN ARRAY ARRAY['position', 'move', 'game', 'match'] LOOP
        IF EXISTS (SELECT 1 FROM pg_class WHERE oid = to_regclass(t) AND relforcerowsecurity) THEN
            EXECUTE format('ALTER TABLE %I NO FORCE ROW LEVEL SECURITY', t);
            INSERT INTO undated_match_forced VALUES (t);
        END IF;
    END LOOP;
END $$;

-- The positions are listed before the matches lose their date, then re-dated
-- from what remains.
CREATE TEMP TABLE undated_match_positions ON COMMIT DROP AS
SELECT DISTINCT mv.tenant_id, mv.position_id
  FROM move mv
  JOIN game g  ON g.id = mv.game_id AND g.tenant_id = mv.tenant_id
  JOIN match m ON m.id = g.match_id AND m.tenant_id = g.tenant_id
 WHERE mv.position_id IS NOT NULL
   AND m.match_date < TIMESTAMPTZ '0002-01-01 00:00:00+00';

UPDATE match SET match_date = NULL
 WHERE match_date < TIMESTAMPTZ '0002-01-01 00:00:00+00';

UPDATE position p SET match_date =
	(SELECT MIN(m.match_date) FROM move mv
	   JOIN game g  ON g.id = mv.game_id AND g.tenant_id = mv.tenant_id
	   JOIN match m ON m.id = g.match_id AND m.tenant_id = g.tenant_id
	  WHERE mv.position_id = p.id AND mv.tenant_id = p.tenant_id)
  FROM undated_match_positions u
 WHERE p.tenant_id = u.tenant_id AND p.id = u.position_id;

DO $$
DECLARE t TEXT;
BEGIN
    FOR t IN SELECT name FROM undated_match_forced LOOP
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
    END LOOP;
END $$;
