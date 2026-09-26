-- Forward migration: the 2.25.0 wave — the Rencontre (ADR-0056).
--
-- A Rencontre is the room several directed Tournaments share: its tables and
-- the output folder of its wall page. What happens in the room — a table out
-- of service, a break — is not stored here: it is an event in the log of
-- every member Direction, so each log still replays on its own.
--
-- direction_pair_member holds the two persons behind a doubles Participant:
-- the engine knows one Participant named "A / B", the Directory needs two.

CREATE TABLE IF NOT EXISTS rencontre (
    id          BIGSERIAL   PRIMARY KEY,
    tenant_id   BIGINT      NOT NULL,
    name        TEXT        NOT NULL,
    starts_on   TEXT        DEFAULT '',
    ends_on     TEXT        DEFAULT '',
    tables      INTEGER     NOT NULL DEFAULT 0,
    output_dir  TEXT        DEFAULT '',
    created_at  TIMESTAMPTZ DEFAULT now(),
    updated_at  TIMESTAMPTZ DEFAULT now(),
    CONSTRAINT rencontre_tenant_id_key UNIQUE (tenant_id, id)
);

ALTER TABLE tournament ADD COLUMN IF NOT EXISTS rencontre_id BIGINT;

-- SET NULL names its column: tenant_id is NOT NULL and must survive the
-- delete (026_set_null_names_its_column.sql). Deleting a Rencontre detaches
-- its Tournaments; it never deletes one.
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'tournament_rencontre_tenant_fkey') THEN
        ALTER TABLE tournament ADD CONSTRAINT tournament_rencontre_tenant_fkey
            FOREIGN KEY (tenant_id, rencontre_id) REFERENCES rencontre (tenant_id, id) ON DELETE SET NULL (rencontre_id);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_tournament_rencontre ON tournament(rencontre_id);

CREATE TABLE IF NOT EXISTS direction_pair_member (
    tournament_id BIGINT  NOT NULL REFERENCES tournament(id) ON DELETE CASCADE,
    tenant_id     BIGINT  NOT NULL,
    player_id     TEXT    NOT NULL,
    seat          INTEGER NOT NULL,
    name          TEXT    NOT NULL,
    club          TEXT    NOT NULL DEFAULT '',
    rating        DOUBLE PRECISION NOT NULL DEFAULT 0,
    PRIMARY KEY (tournament_id, player_id, seat)
);

-- Row-level security, applied only if this schema already has it. Same shape
-- as 025_direction.sql.
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = current_schema() AND c.relname = 'position' AND c.relrowsecurity
    ) THEN
        ALTER TABLE rencontre ENABLE ROW LEVEL SECURITY;
        ALTER TABLE rencontre FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS tenant_isolation ON rencontre;
        CREATE POLICY tenant_isolation ON rencontre
            USING      (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint);

        ALTER TABLE direction_pair_member ENABLE ROW LEVEL SECURITY;
        ALTER TABLE direction_pair_member FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS tenant_isolation ON direction_pair_member;
        CREATE POLICY tenant_isolation ON direction_pair_member
            USING      (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint);
    END IF;
END $$;
