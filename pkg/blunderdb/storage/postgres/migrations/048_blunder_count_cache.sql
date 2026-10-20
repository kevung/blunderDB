-- Forward migration: the 2.44.0 wave — the blunder count is cached.
--
--   * library_settings row (tenant_id, 'blunder_count') holds
--     "<threshold>:<count>" (storage.LibrarySettingBlunderCountKey). Counting
--     reads every analysis; the row spares that scan until a write could
--     change the answer. It lives with the tenant's settings: under RLS,
--     purged with the tenant, never in the global metadata table.
--   * Statement triggers on position, analysis, move and action_label delete
--     the row of every tenant the statement touched, whatever path the write
--     takes. Each first takes a shared advisory lock on that tenant, held to
--     the end of its transaction (blunder_count_share_lock); the reader caches
--     only while holding the same lock exclusively (blunder_count_try_lock),
--     so no write in flight escapes it.
--   * The lock is the two-key form (class, tenant): one lock per tenant, no
--     two tenants sharing one. A tenant id beyond the 32-bit key falls back
--     to the one-key form on the id itself, still one lock per tenant.
--   * TRUNCATE fires no row trigger and empties the table for every tenant
--     at once, past RLS: it is refused on these tables. A tenant's rows go
--     with DELETE (PurgeTenant), which the triggers see.
-- Schema-visible: bumps domain.DatabaseVersion to 2.44.0.

CREATE OR REPLACE FUNCTION blunder_count_share_lock(t BIGINT) RETURNS VOID
LANGUAGE plpgsql AS $$
BEGIN
	IF t BETWEEN 0 AND 2147483647 THEN
		PERFORM pg_advisory_xact_lock_shared(1651273326, t::INTEGER);
	ELSE
		PERFORM pg_advisory_xact_lock_shared(t);
	END IF;
END;
$$;

CREATE OR REPLACE FUNCTION blunder_count_try_lock(t BIGINT) RETURNS BOOLEAN
LANGUAGE plpgsql AS $$
BEGIN
	IF t BETWEEN 0 AND 2147483647 THEN
		RETURN pg_try_advisory_xact_lock(1651273326, t::INTEGER);
	END IF;
	RETURN pg_try_advisory_xact_lock(t);
END;
$$;

CREATE OR REPLACE FUNCTION blunder_count_forget() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
	t BIGINT;
BEGIN
	IF TG_OP = 'TRUNCATE' THEN
		RAISE EXCEPTION 'TRUNCATE % would leave every tenant''s cached blunder count stale: delete the rows instead', TG_TABLE_NAME;
	END IF;
	IF TG_OP = 'INSERT' THEN
		FOR t IN SELECT DISTINCT tenant_id FROM blunder_count_new LOOP
			PERFORM blunder_count_share_lock(t);
			DELETE FROM library_settings WHERE tenant_id = t AND key = 'blunder_count';
		END LOOP;
	ELSIF TG_OP = 'DELETE' THEN
		FOR t IN SELECT DISTINCT tenant_id FROM blunder_count_old LOOP
			PERFORM blunder_count_share_lock(t);
			DELETE FROM library_settings WHERE tenant_id = t AND key = 'blunder_count';
		END LOOP;
	ELSE
		FOR t IN SELECT tenant_id FROM blunder_count_old UNION SELECT tenant_id FROM blunder_count_new LOOP
			PERFORM blunder_count_share_lock(t);
			DELETE FROM library_settings WHERE tenant_id = t AND key = 'blunder_count';
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
		EXECUTE format('CREATE TRIGGER blunder_count_forget_%1$s_truncate BEFORE TRUNCATE ON %1$I
			FOR EACH STATEMENT EXECUTE FUNCTION blunder_count_forget()', tbl);
	END LOOP;
END;
$$;
