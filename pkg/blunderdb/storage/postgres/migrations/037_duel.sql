-- Forward migration: the 2.32.0 wave — the Duel (ADR-0072).
--
--   * duel — the draft of a match played here, under blunderDB's arbitration:
--     one opaque JSON document rewritten after every Action, like
--     transcription, and the dice seed in a column of its own, written at
--     creation and never again. Every roll is computed from the seed, which
--     leaves the Arbiter only with the finished Match.
--   * match_origin — how a Match played here came to be: its Start (an XGID,
--     '' for the opening position), the revealed seed, whether it was stopped
--     before its end or lost on time, the Bot's level and the Cadence. A Match
--     without a row was not played here. It goes with its Match: the composite
--     foreign key targets the (tenant_id, id) key of match.
-- Schema-visible: bumps domain.DatabaseVersion to 2.32.0.

CREATE TABLE IF NOT EXISTS duel (
    id             BIGSERIAL   PRIMARY KEY,
    tenant_id      BIGINT      NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    format_version TEXT        NOT NULL,
    label          TEXT        NOT NULL DEFAULT '',
    document       TEXT        NOT NULL,
    dice_seed      TEXT        NOT NULL,
    revision       BIGINT      NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS match_origin (
    match_id      BIGINT  PRIMARY KEY,
    tenant_id     BIGINT  NOT NULL,
    start         TEXT    NOT NULL DEFAULT '',
    dice_seed     TEXT    NOT NULL,
    stopped_early INTEGER NOT NULL DEFAULT 0,
    lost_on_time  INTEGER NOT NULL DEFAULT 0,
    bot_level     TEXT    NOT NULL DEFAULT '',
    cadence       TEXT    NOT NULL DEFAULT '',
    CONSTRAINT match_origin_match_tenant_fkey
        FOREIGN KEY (tenant_id, match_id) REFERENCES match (tenant_id, id) ON DELETE CASCADE
);

-- Row-level security, applied only if this schema already has it. Same shape
-- as 033_met_progress_move_error.sql.
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = current_schema() AND c.relname = 'position' AND c.relrowsecurity
    ) THEN
        ALTER TABLE duel ENABLE ROW LEVEL SECURITY;
        ALTER TABLE duel FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS tenant_isolation ON duel;
        CREATE POLICY tenant_isolation ON duel
            USING      (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint);
        ALTER TABLE match_origin ENABLE ROW LEVEL SECURITY;
        ALTER TABLE match_origin FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS tenant_isolation ON match_origin;
        CREATE POLICY tenant_isolation ON match_origin
            USING      (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint);
    END IF;
END $$;
