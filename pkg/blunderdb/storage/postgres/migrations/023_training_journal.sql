-- Forward migration: the 2.22.0 wave — the Training journal (issue #320).
--
-- Two tables, not a JSON key of `metadata`: the per-number detail is the whole
-- point of the journal ("tp4 last roll: 6 faults in 9"), and a blob that grows
-- by one entry per revealed number is a register a table settles (ADR-0040
-- rule 6). No cap, for the same reason: the fifty-session bound the old
-- metadata key carried existed to keep a metadata VALUE small.
--
-- `deviations` is not redundant with `mean_deviation`. A declared exercise
-- (Scores, Pips) produces no deviation at all, and a question that ran out of
-- time produces none either — averaging its missing answer as a zero error
-- would flatter the mean. The count says how many numbers the mean is over.

CREATE TABLE IF NOT EXISTS training_session (
    id             BIGSERIAL PRIMARY KEY,
    tenant_id      BIGINT           NOT NULL,
    -- scores | pips | bearoff | evaluation | decision
    exercise       TEXT             NOT NULL,
    -- pool | board | library (ADR-0041 rule 2); empty when the exercise has
    -- only one source.
    seed_source    TEXT             NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ      NOT NULL DEFAULT now(),
    numbers_asked  INTEGER          NOT NULL DEFAULT 0,
    faults         INTEGER          NOT NULL DEFAULT 0,
    deviations     INTEGER          NOT NULL DEFAULT 0,
    mean_deviation DOUBLE PRECISION NOT NULL DEFAULT 0,
    median_ms      INTEGER          NOT NULL DEFAULT 0,
    -- The session PR, for the Decision exercise only; 0 elsewhere.
    pr             DOUBLE PRECISION NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_training_session_exercise ON training_session (tenant_id, exercise, id);

-- The composite (tenant_id, id) key training_item's foreign key points at, so
-- a row can never reference a session of another tenant (017_composite_tenant_fk).
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'training_session_tenant_id_key') THEN
        ALTER TABLE training_session ADD CONSTRAINT training_session_tenant_id_key UNIQUE (tenant_id, id);
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS training_item (
    id            BIGSERIAL PRIMARY KEY,
    tenant_id     BIGINT           NOT NULL,
    session_id    BIGINT           NOT NULL,
    -- The kind of number, never the face it was asked from: the same cell of
    -- the same table seen from either side is one weakness, not two
    -- (ADR-0040 rule 4). "tp4.last", "gv2", "pips.bottom", …
    number_type   TEXT             NOT NULL,
    wrong         BOOLEAN          NOT NULL DEFAULT FALSE,
    has_deviation BOOLEAN          NOT NULL DEFAULT FALSE,
    deviation     DOUBLE PRECISION NOT NULL DEFAULT 0
);

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'training_item_session_fkey') THEN
        ALTER TABLE training_item ADD CONSTRAINT training_item_session_fkey
            FOREIGN KEY (tenant_id, session_id)
            REFERENCES training_session (tenant_id, id) ON DELETE CASCADE;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_training_item_session ON training_item (tenant_id, session_id);
CREATE INDEX IF NOT EXISTS idx_training_item_type ON training_item (tenant_id, number_type);

-- Two new tenant-scoped tables need the isolation policy the rest of the
-- schema carries, and need it only where RLS is already enforced (a database
-- bootstrapped without it must not silently gain half of it). Same shape as
-- 019_product_wave_2_19_0.sql.
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = current_schema() AND c.relname = 'position' AND c.relrowsecurity
    ) THEN
        ALTER TABLE training_session ENABLE ROW LEVEL SECURITY;
        ALTER TABLE training_session FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS tenant_isolation ON training_session;
        CREATE POLICY tenant_isolation ON training_session
            USING      (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint);

        ALTER TABLE training_item ENABLE ROW LEVEL SECURITY;
        ALTER TABLE training_item FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS tenant_isolation ON training_item;
        CREATE POLICY tenant_isolation ON training_item
            USING      (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint);
    END IF;
END $$;
