-- Forward migration: the 2.44.0 wave — the blunder count is cached.
--
--   * metadata row 'blunder_count:<tenant>' holds "<threshold>:<count>"
--     (storage.BlunderCountMetadataKey). Counting reads every analysis;
--     the row spares that scan until a write could change the answer.
--   * Statement triggers on position, analysis, move and action_label
--     delete the row of every tenant the statement touched, whatever path
--     the write takes. Each first takes a shared advisory lock on the tenant
--     (class 1651273326, sqlshared.BlunderCountLockClass), held to the end of
--     its transaction: the count is cached only by a reader that holds the
--     same lock exclusively, so no write in flight can be missed by it.
--   * TRUNCATE fires no row trigger: its own trigger deletes every tenant's
--     row (it waits on the readers' table locks, so none caches past it).
-- Schema-visible: bumps domain.DatabaseVersion to 2.44.0.

CREATE OR REPLACE FUNCTION blunder_count_forget() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
	t BIGINT;
BEGIN
	IF TG_OP = 'TRUNCATE' THEN
		DELETE FROM metadata WHERE key LIKE 'blunder\_count:%';
		RETURN NULL;
	END IF;
	IF TG_OP = 'INSERT' THEN
		FOR t IN SELECT DISTINCT tenant_id FROM blunder_count_new LOOP
			PERFORM pg_advisory_xact_lock_shared(1651273326, CAST(MOD(t, 2147483647) AS INTEGER));
			DELETE FROM metadata WHERE key = 'blunder_count:' || t;
		END LOOP;
	ELSIF TG_OP = 'DELETE' THEN
		FOR t IN SELECT DISTINCT tenant_id FROM blunder_count_old LOOP
			PERFORM pg_advisory_xact_lock_shared(1651273326, CAST(MOD(t, 2147483647) AS INTEGER));
			DELETE FROM metadata WHERE key = 'blunder_count:' || t;
		END LOOP;
	ELSE
		FOR t IN SELECT tenant_id FROM blunder_count_old UNION SELECT tenant_id FROM blunder_count_new LOOP
			PERFORM pg_advisory_xact_lock_shared(1651273326, CAST(MOD(t, 2147483647) AS INTEGER));
			DELETE FROM metadata WHERE key = 'blunder_count:' || t;
		END LOOP;
	END IF;
	RETURN NULL;
END;
$$;

DO $$
DECLARE
	tbl TEXT;
BEGIN
	FOREACH tbl IN ARRAY ARRAY['position', 'analysis', 'move', 'action_label'] LOOP
		EXECUTE format('DROP TRIGGER IF EXISTS blunder_count_forget_%1$s_insert ON %1$I', tbl);
		EXECUTE format('CREATE TRIGGER blunder_count_forget_%1$s_insert AFTER INSERT ON %1$I
			REFERENCING NEW TABLE AS blunder_count_new
			FOR EACH STATEMENT EXECUTE FUNCTION blunder_count_forget()', tbl);
		EXECUTE format('DROP TRIGGER IF EXISTS blunder_count_forget_%1$s_update ON %1$I', tbl);
		EXECUTE format('CREATE TRIGGER blunder_count_forget_%1$s_update AFTER UPDATE ON %1$I
			REFERENCING OLD TABLE AS blunder_count_old NEW TABLE AS blunder_count_new
			FOR EACH STATEMENT EXECUTE FUNCTION blunder_count_forget()', tbl);
		EXECUTE format('DROP TRIGGER IF EXISTS blunder_count_forget_%1$s_delete ON %1$I', tbl);
		EXECUTE format('CREATE TRIGGER blunder_count_forget_%1$s_delete AFTER DELETE ON %1$I
			REFERENCING OLD TABLE AS blunder_count_old
			FOR EACH STATEMENT EXECUTE FUNCTION blunder_count_forget()', tbl);
		EXECUTE format('DROP TRIGGER IF EXISTS blunder_count_forget_%1$s_truncate ON %1$I', tbl);
		EXECUTE format('CREATE TRIGGER blunder_count_forget_%1$s_truncate AFTER TRUNCATE ON %1$I
			FOR EACH STATEMENT EXECUTE FUNCTION blunder_count_forget()', tbl);
	END LOOP;
END;
$$;
