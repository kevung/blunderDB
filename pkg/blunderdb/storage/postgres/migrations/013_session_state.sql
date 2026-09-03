-- Forward migration: the UI session state (last search, last position, open
-- views) leaves the global metadata table for its own tenant-scoped table
-- (issue #156, schema 2.17.0).
--
-- Until 2.16.0 sqlshared.SessionStore wrote a tenant's session as six metadata
-- rows keyed '<scope>:session_*' (bare 'session_*' for the empty scope). The
-- metadata table has no tenant_id, sits outside Row-Level Security and was
-- readable whole through /v1/metadata.load — so every tenant could read every
-- other tenant's session, and could write database_version. The route is gone;
-- session_state carries tenant_id like every other domain table, is covered by
-- PurgeTenant and by the RLS policy set (rls_postgres.go), and metadata is left
-- holding database infrastructure only.
--
-- Rows are moved for every scope that is a tenant as ADR-0005 (amendment
-- 2026-09-03) defines it: a positive decimal integer. A prefix that is not
-- one ('alice:session_views', written before that amendment) names a tenant
-- the daemon no longer accepts and whose session — six lines of UI state —
-- is unreachable; those rows are dropped with the rest so that metadata holds
-- no per-tenant data at all.
--
-- Idempotent: IF NOT EXISTS, upsert on copy, and the DELETE finds nothing on a
-- second run or on a fresh database whose 001 baseline already has the table.

CREATE TABLE IF NOT EXISTS session_state (
    tenant_id  BIGINT NOT NULL,
    key        TEXT   NOT NULL,
    value      TEXT,
    PRIMARY KEY (tenant_id, key)
);

-- '<n>:session_*' rows: tenant n.
INSERT INTO session_state (tenant_id, key, value)
SELECT split_part(key, ':session_', 1)::bigint,
       substr(key, position(':session_' IN key) + 1),
       value
FROM metadata
WHERE position(':session_' IN key) > 0
  AND split_part(key, ':session_', 1) ~ '^[1-9][0-9]*$'
ON CONFLICT (tenant_id, key) DO UPDATE SET value = EXCLUDED.value;

-- Bare 'session_*' rows: the empty scope, tenant 0.
INSERT INTO session_state (tenant_id, key, value)
SELECT 0, key, value
FROM metadata
WHERE key LIKE 'session\_%'
ON CONFLICT (tenant_id, key) DO UPDATE SET value = EXCLUDED.value;

DELETE FROM metadata
WHERE key LIKE 'session\_%' OR position(':session_' IN key) > 0;

-- A database that already enforces RLS (serve --rls ran ApplyRLS on it) must
-- not gain an unprotected table between this migration and the next ApplyRLS:
-- install the same fail-closed policy now. On a database without RLS this is
-- a no-op — enabling it on one table only would hide every session from a
-- daemon that sets no app.tenant_id.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = current_schema() AND c.relname = 'position' AND c.relrowsecurity
    ) THEN
        ALTER TABLE session_state ENABLE ROW LEVEL SECURITY;
        ALTER TABLE session_state FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS tenant_isolation ON session_state;
        CREATE POLICY tenant_isolation ON session_state
            USING      (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint);
    END IF;
END $$;

