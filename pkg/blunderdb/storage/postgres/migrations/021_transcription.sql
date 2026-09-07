-- Forward migration: the 2.21.0 wave — the `transcription` table (issue #334).
--
-- A transcription is a DRAFT (ADR-0045 §1): the match a user is typing in,
-- before it is worth a row in match/game/move. Writing it straight into the
-- match graph would put every keystroke through the position dedup and the
-- orphan purge, and would count a half-typed match in the Performance Rating.
--
-- The row is one opaque JSON `document` plus the handful of columns the
-- library list needs to show a draft without parsing it. The document carries
-- its OWN `format_version`, so a change to its shape is a version of the
-- document and never a domain.DatabaseVersion migration — this table is
-- expected to be created once and never altered again.
--
-- match_id is the Match the draft has already produced, NULL while it has
-- produced none. ON DELETE SET NULL rather than CASCADE: deleting the saved
-- match must not destroy the typing that produced it — the draft simply
-- becomes one that was never saved. The composite FK targets the
-- (tenant_id, id) key 017_composite_tenant_fk.sql put on `match`, so a draft
-- can only ever point at a match of its own tenant.

CREATE TABLE IF NOT EXISTS transcription (
    id             BIGSERIAL PRIMARY KEY,
    tenant_id      BIGINT      NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    format_version TEXT        NOT NULL,
    match_id       BIGINT,
    label          TEXT        NOT NULL DEFAULT '',
    document       TEXT        NOT NULL
);

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'transcription_match_fkey') THEN
        ALTER TABLE transcription ADD CONSTRAINT transcription_match_fkey
            FOREIGN KEY (tenant_id, match_id)
            REFERENCES match (tenant_id, id) ON DELETE SET NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_transcription_match ON transcription (tenant_id, match_id);

-- The new tenant-scoped table needs the isolation policy the rest of the
-- schema carries, and needs it only where RLS is already enforced (a database
-- bootstrapped without it must not silently gain half of it). Same shape as
-- 019_product_wave_2_19_0.sql.
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = current_schema() AND c.relname = 'position' AND c.relrowsecurity
    ) THEN
        ALTER TABLE transcription ENABLE ROW LEVEL SECURITY;
        ALTER TABLE transcription FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS tenant_isolation ON transcription;
        CREATE POLICY tenant_isolation ON transcription
            USING      (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint);
    END IF;
END $$;
