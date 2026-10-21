-- Forward migration: the 2.49.0 wave — position.multi_played.
--
--   * multi_played — player 1 recorded two distinct plays on the position
--                    (checker move or cube action, compared as stored):
--                    the error columns of its analysis score one play, so a
--                    move-error search re-scores these rows in Go. The same
--                    definition as sqlshared.MultiPlayedSQL, with a tenant.
--   * Statement triggers on move keep it on every write, whatever path the
--     write takes: an insert can only set it, a delete only clear it, an
--     update re-reads the positions whose plays it changed. An update that
--     leaves position, player and play alone (the error columns, rewritten
--     with every analysis) re-reads nothing.
--   * The rows already written are filled here. Row-Level Security, once
--     applied, is FORCEd on position and move: the migrating connection
--     carries no tenant and would see no row, so FORCE is lifted for the
--     fill and put back; the batch runs as one transaction.
-- Idempotent: a second run rewrites the same values.
-- Schema-visible: bumps domain.DatabaseVersion to 2.49.0.

ALTER TABLE position ADD COLUMN IF NOT EXISTS multi_played BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_position_multi_played ON position (tenant_id, id) WHERE multi_played;

-- cube_action is read as text, a NULL as the code 0: the same comparison as
-- on integer codes, and it still runs on a library whose cube_action is the
-- TEXT column 034 converts, should a write reach move before 034 has run.
CREATE OR REPLACE FUNCTION multi_played_of(t BIGINT, pid BIGINT) RETURNS BOOLEAN
LANGUAGE sql STABLE AS $$
	SELECT (SELECT COUNT(*) FROM (
		SELECT DISTINCT COALESCE(checker_move, ''), COALESCE(cube_action::text, '0')
		  FROM move
		 WHERE tenant_id = t AND position_id = pid AND player = 1
		 LIMIT 2) d) > 1
$$;

CREATE OR REPLACE FUNCTION multi_played_refresh() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
	IF TG_OP = 'INSERT' THEN
		UPDATE position p SET multi_played = TRUE
		  FROM (SELECT DISTINCT tenant_id, position_id FROM multi_played_new
		         WHERE player = 1 AND position_id IS NOT NULL) c
		 WHERE p.tenant_id = c.tenant_id AND p.id = c.position_id
		   AND NOT p.multi_played AND multi_played_of(c.tenant_id, c.position_id);
	ELSIF TG_OP = 'DELETE' THEN
		UPDATE position p SET multi_played = FALSE
		  FROM (SELECT DISTINCT tenant_id, position_id FROM multi_played_old
		         WHERE player = 1 AND position_id IS NOT NULL) c
		 WHERE p.tenant_id = c.tenant_id AND p.id = c.position_id
		   AND p.multi_played AND NOT multi_played_of(c.tenant_id, c.position_id);
	ELSE
		UPDATE position p SET multi_played = multi_played_of(c.tenant_id, c.position_id)
		  FROM (SELECT o.tenant_id, o.position_id AS old_id, n.position_id AS new_id
		          FROM multi_played_old o
		          JOIN multi_played_new n ON n.tenant_id = o.tenant_id AND n.id = o.id
		         WHERE (o.player = 1 OR n.player = 1)
		           AND (o.position_id, o.player, o.checker_move, o.cube_action)
		               IS DISTINCT FROM (n.position_id, n.player, n.checker_move, n.cube_action)) ch
		 CROSS JOIN LATERAL (VALUES (ch.tenant_id, ch.old_id), (ch.tenant_id, ch.new_id)) AS c(tenant_id, position_id)
		 WHERE p.tenant_id = c.tenant_id AND p.id = c.position_id
		   AND p.multi_played IS DISTINCT FROM multi_played_of(c.tenant_id, c.position_id);
	END IF;
	RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS multi_played_move_insert ON move;
CREATE TRIGGER multi_played_move_insert AFTER INSERT ON move
	REFERENCING NEW TABLE AS multi_played_new
	FOR EACH STATEMENT EXECUTE FUNCTION multi_played_refresh();
DROP TRIGGER IF EXISTS multi_played_move_update ON move;
CREATE TRIGGER multi_played_move_update AFTER UPDATE ON move
	REFERENCING OLD TABLE AS multi_played_old NEW TABLE AS multi_played_new
	FOR EACH STATEMENT EXECUTE FUNCTION multi_played_refresh();
DROP TRIGGER IF EXISTS multi_played_move_delete ON move;
CREATE TRIGGER multi_played_move_delete AFTER DELETE ON move
	REFERENCING OLD TABLE AS multi_played_old
	FOR EACH STATEMENT EXECUTE FUNCTION multi_played_refresh();

CREATE TEMP TABLE multi_played_forced (name TEXT) ON COMMIT DROP;

DO $$
DECLARE t TEXT;
BEGIN
    FOREACH t IN ARRAY ARRAY['position', 'move'] LOOP
        IF EXISTS (SELECT 1 FROM pg_class WHERE oid = to_regclass(t) AND relforcerowsecurity) THEN
            EXECUTE format('ALTER TABLE %I NO FORCE ROW LEVEL SECURITY', t);
            INSERT INTO multi_played_forced VALUES (t);
        END IF;
    END LOOP;
END $$;

UPDATE position p SET multi_played = (c.position_id IS NOT NULL)
  FROM position q
  LEFT JOIN (SELECT tenant_id, position_id FROM move
              WHERE player = 1 AND position_id IS NOT NULL
              GROUP BY tenant_id, position_id
             HAVING COUNT(DISTINCT COALESCE(checker_move, '') || '|' || COALESCE(cube_action, 0)) > 1) c
    ON c.tenant_id = q.tenant_id AND c.position_id = q.id
 WHERE p.tenant_id = q.tenant_id AND p.id = q.id
   AND p.multi_played IS DISTINCT FROM (c.position_id IS NOT NULL);

DO $$
DECLARE t TEXT;
BEGIN
    FOR t IN SELECT name FROM multi_played_forced LOOP
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
    END LOOP;
END $$;
