-- Forward migration: the 2.19.0 product wave (lot I of the 2026-09b plan).
--
-- Four changes in one migration, because they are one schema version:
--   * comment.origin        — who wrote a comment (issue #263);
--   * import_batch (+ match.import_batch_id) — the end-of-import report's
--     unit of account (issue #257);
--   * trash                 — the snapshot table undo restores from (#285);
--   * position.game_phase   — added by 018, listed here only so the wave is
--     readable as one thing.
--
-- comment.origin defaults to 'unknown' and NOT to 'user'. A row that predates
-- the column has no provenance, and calling it the user's would make the
-- retention predicate (matches_postgres.go, positionIsHeldSQL) start sparing
-- positions it has always purged — a silent change of behaviour on databases
-- nobody touched.

ALTER TABLE comment ADD COLUMN IF NOT EXISTS origin TEXT NOT NULL DEFAULT 'unknown';

DROP INDEX IF EXISTS idx_comment_position;
CREATE INDEX IF NOT EXISTS idx_comment_position ON comment (tenant_id, position_id, origin);

CREATE TABLE IF NOT EXISTS import_batch (
    id          BIGSERIAL PRIMARY KEY,
    tenant_id   BIGINT      NOT NULL,
    started_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    source      TEXT        NOT NULL DEFAULT '',
    format      TEXT        NOT NULL DEFAULT '',
    counts      TEXT        NOT NULL DEFAULT '{}'
);

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'import_batch_tenant_id_key') THEN
        ALTER TABLE import_batch ADD CONSTRAINT import_batch_tenant_id_key UNIQUE (tenant_id, id);
    END IF;
END $$;

ALTER TABLE match ADD COLUMN IF NOT EXISTS import_batch_id BIGINT;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'match_import_batch_fkey') THEN
        ALTER TABLE match ADD CONSTRAINT match_import_batch_fkey
            FOREIGN KEY (tenant_id, import_batch_id)
            REFERENCES import_batch (tenant_id, id) ON DELETE SET NULL;
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS trash (
    id         BIGSERIAL PRIMARY KEY,
    tenant_id  BIGINT      NOT NULL,
    kind       TEXT        NOT NULL,
    label      TEXT        NOT NULL DEFAULT '',
    payload    TEXT        NOT NULL,
    deleted_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_trash_deleted_at ON trash (tenant_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_trash_kind ON trash (tenant_id, kind, deleted_at);

-- Two new tenant-scoped tables need the isolation policy the rest of the
-- schema carries, and need it only where RLS is already enforced (a database
-- bootstrapped without it must not silently gain half of it). Same shape as
-- 013_session_state.sql.
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = current_schema() AND c.relname = 'position' AND c.relrowsecurity
    ) THEN
        ALTER TABLE import_batch ENABLE ROW LEVEL SECURITY;
        ALTER TABLE import_batch FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS tenant_isolation ON import_batch;
        CREATE POLICY tenant_isolation ON import_batch
            USING      (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint);

        ALTER TABLE trash ENABLE ROW LEVEL SECURITY;
        ALTER TABLE trash FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS tenant_isolation ON trash;
        CREATE POLICY tenant_isolation ON trash
            USING      (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint);
    END IF;
END $$;
