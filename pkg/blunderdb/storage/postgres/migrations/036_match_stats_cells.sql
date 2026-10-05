-- Forward migration: the per-match breakdowns beside match_stats.
--
-- match_stats_cell holds a seat's counted decisions grouped by one dimension
-- per kind (phase, plan of play, score, cube actions, error bucket), and
-- match_stats_position the shared positions they reach; the arithmetic and
-- the kinds are stated once, in sqlshared/match_stats_cells.go, and the
-- SQLite declaration is schema_sqlite.go's. Both reference their seat's
-- match_stats row ON DELETE CASCADE: every invalidation of match_stats drops
-- them too. A match_stats row written before them is told apart and
-- recomputed at open (sqlshared.DropOlderShapeMatchStatsSQL). Part of the
-- 2.31.0 schema; this file does not write database_version.

CREATE TABLE IF NOT EXISTS match_stats_cell (
    tenant_id     BIGINT  NOT NULL,
    match_id      BIGINT  NOT NULL,
    seat          INTEGER NOT NULL,
    decision_type INTEGER NOT NULL,
    met_id        BIGINT  NOT NULL,
    kind          INTEGER NOT NULL,
    k1            BIGINT  NOT NULL,
    k2            BIGINT  NOT NULL,
    decisions     BIGINT  NOT NULL,
    error_mp      BIGINT  NOT NULL,
    max_error_mp  BIGINT  NOT NULL,
    blunders      BIGINT  NOT NULL,
    mwc_loss      DOUBLE PRECISION NOT NULL,
    mwc_decisions BIGINT  NOT NULL,
    positions     BIGINT  NOT NULL,
    PRIMARY KEY (tenant_id, match_id, seat, decision_type, met_id, kind, k1, k2),
    CONSTRAINT match_stats_cell_seat_fkey
        FOREIGN KEY (tenant_id, match_id, seat) REFERENCES match_stats (tenant_id, match_id, seat) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS match_stats_position (
    tenant_id     BIGINT  NOT NULL,
    match_id      BIGINT  NOT NULL,
    seat          INTEGER NOT NULL,
    position_id   BIGINT  NOT NULL,
    decision_type INTEGER NOT NULL,
    met_id        BIGINT  NOT NULL,
    PRIMARY KEY (tenant_id, match_id, seat, position_id, met_id),
    CONSTRAINT match_stats_position_seat_fkey
        FOREIGN KEY (tenant_id, match_id, seat) REFERENCES match_stats (tenant_id, match_id, seat) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_match_stats_position_position ON match_stats_position (tenant_id, position_id);

-- Row-level security, applied only if this schema already has it. Same shape
-- as 035_study_mark.sql.
DO $$
DECLARE t TEXT;
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = current_schema() AND c.relname = 'position' AND c.relrowsecurity
    ) THEN
        FOREACH t IN ARRAY ARRAY['match_stats_cell', 'match_stats_position'] LOOP
            EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
            EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
            EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', t);
            EXECUTE format('CREATE POLICY tenant_isolation ON %I
                USING      (tenant_id = NULLIF(current_setting(''app.tenant_id'', true), '''')::bigint)
                WITH CHECK (tenant_id = NULLIF(current_setting(''app.tenant_id'', true), '''')::bigint)', t);
        END LOOP;
    END IF;
END $$;
