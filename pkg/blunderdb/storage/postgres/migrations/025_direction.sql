-- Forward migration: the 2.24.0 wave — the Direction of a Tournament (issue #365, ADR-0047).
--
-- A Tournament used to be an afterthought: a label put on Matches that came in
-- from files. It can now be created BEFORE its Matches exist and run from here,
-- and everything the director decides is kept as an event log.
--
-- Two rules shape these tables, and both come from ADR-0047:
--
--   1. The log is APPEND-ONLY. A wrong result is corrected by a later
--      correction event, a match launched by mistake by a later cancellation.
--      Nothing here is ever UPDATEd or DELETEd outside the cascade from a
--      deleted Tournament.
--   2. The derived state — standings, brackets, pairings, what to do next — is
--      NEVER stored. It is replayed from the events at every open. That is what
--      makes a power cut in the middle of a tournament cost nothing, and what
--      keeps a correction from leaving a stale intermediate state behind.
--
-- `config` holds the configuration only while the Direction is in preparation
-- (state = 'draft'); the first launched match freezes it into the `created`
-- event and this column stops being read.

CREATE TABLE IF NOT EXISTS direction (
    tournament_id  BIGINT      PRIMARY KEY REFERENCES tournament(id) ON DELETE CASCADE,
    tenant_id      BIGINT      NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Version of the Nicomaque journal format this Direction was written with.
    format_version INTEGER     NOT NULL DEFAULT 1,
    -- Tag of the engine module, for the info button and for diagnosing a replay.
    engine_version TEXT        NOT NULL DEFAULT '',
    -- draft | running | finished
    state          TEXT        NOT NULL DEFAULT 'draft',
    config         TEXT        NOT NULL DEFAULT '',
    output_dir     TEXT        NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS direction_event (
    tournament_id BIGINT      NOT NULL REFERENCES tournament(id) ON DELETE CASCADE,
    tenant_id     BIGINT      NOT NULL,
    seq           INTEGER     NOT NULL,
    kind          TEXT        NOT NULL,
    time          TIMESTAMPTZ NOT NULL,
    payload       TEXT        NOT NULL,
    PRIMARY KEY (tournament_id, seq)
);

CREATE INDEX IF NOT EXISTS idx_direction_event_tournament ON direction_event(tournament_id, seq);

-- The Slot this Match fills in its Tournament's Direction: the Nicomaque match
-- id ("M12"), empty when the Match fills no Slot. A Match fills at most one
-- Slot and a Slot carries at most one Match.
ALTER TABLE match ADD COLUMN IF NOT EXISTS direction_match_id TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_match_direction_slot ON match(tournament_id, direction_match_id) WHERE direction_match_id <> '';

-- Row-level security, applied only if this schema already has it (a schema
-- bootstrapped without RLS must not silently gain half of it). Same shape as
-- 024_anki_score_cards.sql.
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = current_schema() AND c.relname = 'position' AND c.relrowsecurity
    ) THEN
        ALTER TABLE direction ENABLE ROW LEVEL SECURITY;
        ALTER TABLE direction FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS tenant_isolation ON direction;
        CREATE POLICY tenant_isolation ON direction
            USING      (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint);

        ALTER TABLE direction_event ENABLE ROW LEVEL SECURITY;
        ALTER TABLE direction_event FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS tenant_isolation ON direction_event;
        CREATE POLICY tenant_isolation ON direction_event
            USING      (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint);
    END IF;
END $$;
