-- Forward migration: the library's own settings — the error and blunder
-- thresholds — get a tenant-scoped table (ADR-0043).
--
-- On the desktop the two rows live in the SQLite file's metadata table, next
-- to the Performance Rating objective: metadata there IS the library. Here it
-- is not. The PostgreSQL metadata table is database infrastructure shared by
-- every tenant, outside Row-Level Security, and its load/save routes were
-- removed after #156 precisely so no tenant reads another's state through it.
-- A threshold is per-tenant data, so it lands where per-tenant data goes:
-- its own table, carrying tenant_id, covered by the RLS policy set
-- (rls_postgres.go) and by PurgeTenant (purge_postgres.go).
--
-- No row is created here. A tenant that has never set a threshold has no row
-- and reads the defaults (50 and 100 millipoints), which are the values every
-- consumer used as constants before this migration — so nothing counted
-- differently the day it ran.
--
-- The shape is session_state's, deliberately: a key/value pair per tenant, the
-- form this schema already uses for the settings a tenant owns.
--
-- Idempotent: IF NOT EXISTS on the table and on the index.

CREATE TABLE IF NOT EXISTS library_settings (
    tenant_id  BIGINT NOT NULL,
    key        TEXT   NOT NULL,
    value      TEXT,
    PRIMARY KEY (tenant_id, key)
);
