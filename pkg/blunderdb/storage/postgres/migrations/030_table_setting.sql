-- Forward migration: the 2.28.0 wave — the properties of a table and the
-- rooms of an event (ADR-0058).
--
-- table_setting holds one row per table that has properties: its number, the
-- identity players and the CLI use, then name, room, reserved and the persons
-- it is kept for (a JSON array of names). Its owner is the Rencontre whose
-- events share the tables or a Tournament run on its own, never both. A room
-- is only the label several rows share: it has no row of its own.
--
-- tournament.rencontre_rooms is the JSON array of rooms an event may play in
-- within its Rencontre; NULL plays on every table. It belongs to the
-- membership, next to rencontre_id, and is cleared when the event is detached.
-- Schema-visible: bumps domain.DatabaseVersion to 2.28.0.

ALTER TABLE tournament ADD COLUMN IF NOT EXISTS rencontre_rooms TEXT;

CREATE TABLE IF NOT EXISTS table_setting (
    id            BIGSERIAL PRIMARY KEY,
    tenant_id     BIGINT    NOT NULL,
    rencontre_id  BIGINT,
    tournament_id BIGINT,
    number        INTEGER   NOT NULL CHECK (number > 0),
    name          TEXT      NOT NULL DEFAULT '',
    room          TEXT      NOT NULL DEFAULT '',
    reserved      BOOLEAN   NOT NULL DEFAULT FALSE,
    assigned_to   TEXT      NOT NULL DEFAULT '[]',
    CONSTRAINT table_setting_one_owner CHECK ((rencontre_id IS NULL) <> (tournament_id IS NULL)),
    -- Composite keys: a row can only belong to an owner of its own tenant
    -- (017_composite_tenant_fk.sql).
    CONSTRAINT table_setting_rencontre_tenant_fkey
        FOREIGN KEY (tenant_id, rencontre_id) REFERENCES rencontre (tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT table_setting_tournament_tenant_fkey
        FOREIGN KEY (tenant_id, tournament_id) REFERENCES tournament (tenant_id, id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_table_setting_rencontre ON table_setting(rencontre_id, number) WHERE rencontre_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_table_setting_tournament ON table_setting(tournament_id, number) WHERE tournament_id IS NOT NULL;

-- Row-level security, applied only if this schema already has it. Same shape
-- as 027_rencontre.sql.
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = current_schema() AND c.relname = 'position' AND c.relrowsecurity
    ) THEN
        ALTER TABLE table_setting ENABLE ROW LEVEL SECURITY;
        ALTER TABLE table_setting FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS tenant_isolation ON table_setting;
        CREATE POLICY tenant_isolation ON table_setting
            USING      (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint);
    END IF;
END $$;
