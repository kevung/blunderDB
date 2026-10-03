-- Forward migration: the 2.30.0 wave — the large-library schema.
--
--   * analysis.analysis_engine / analysis_depth / creation_date — who gave
--     the verdict, how deep (domain.AnalysisDepthRank) and when the blob says
--     it was written (engine.AnalysisProvenance). NULL analysis_engine means
--     "not derived yet": the blob is compressed JSON SQL cannot read, so
--     backfillAnalysisProvenance (postgres.go) derives them in Go after the
--     forward chain.
--   * position.match_date — the date of the earliest match that reaches the
--     position, kept true by the match store; derived here, set-based.
--   * import_batch_file — one row per file an import batch met.
--   * player_alias / event_alias — other spellings of one name.
--   * Index pruning, decided on the query plans of a 15.6 M-position library
--     (tasks/search-query-plans.txt): no index leads with decision_type, the
--     cube-response index keeps only its selective side, and the player-2
--     rates get the covering index the player-1 rates have.
-- Schema-visible: bumps domain.DatabaseVersion to 2.30.0.

ALTER TABLE analysis ADD COLUMN IF NOT EXISTS analysis_engine TEXT;
ALTER TABLE analysis ADD COLUMN IF NOT EXISTS analysis_depth  INTEGER;
ALTER TABLE analysis ADD COLUMN IF NOT EXISTS creation_date   TIMESTAMPTZ;
ALTER TABLE position ADD COLUMN IF NOT EXISTS match_date      TIMESTAMPTZ;

UPDATE position p SET match_date = t.md
FROM (SELECT mv.tenant_id, mv.position_id, MIN(m.match_date) AS md
        FROM move mv
        JOIN game g  ON g.id = mv.game_id AND g.tenant_id = mv.tenant_id
        JOIN match m ON m.id = g.match_id AND m.tenant_id = g.tenant_id
       WHERE mv.position_id IS NOT NULL
       GROUP BY mv.tenant_id, mv.position_id) t
WHERE p.tenant_id = t.tenant_id AND p.id = t.position_id AND p.match_date IS NULL;

DROP INDEX IF EXISTS idx_position_decision_dice;
DROP INDEX IF EXISTS idx_position_decision_pip;
DROP INDEX IF EXISTS idx_position_cube_response;
DROP INDEX IF EXISTS idx_analysis_win2;
DROP INDEX IF EXISTS idx_analysis_gammon2;

CREATE INDEX IF NOT EXISTS idx_position_cube_take ON position (tenant_id, is_cube_response) WHERE is_cube_response;
CREATE INDEX IF NOT EXISTS idx_analysis_win_gammon2_covering ON analysis (tenant_id, player2_win_rate, player2_gammon_rate, position_id);
CREATE INDEX IF NOT EXISTS idx_analysis_engine        ON analysis (tenant_id, analysis_engine);
CREATE INDEX IF NOT EXISTS idx_analysis_depth         ON analysis (tenant_id, analysis_depth);
CREATE INDEX IF NOT EXISTS idx_analysis_creation_date ON analysis (tenant_id, creation_date);
CREATE INDEX IF NOT EXISTS idx_position_match_date    ON position (tenant_id, match_date);

CREATE TABLE IF NOT EXISTS import_batch_file (
    id        BIGSERIAL   PRIMARY KEY,
    tenant_id BIGINT      NOT NULL,
    batch_id  BIGINT      NOT NULL,
    path      TEXT        NOT NULL,
    size      BIGINT      NOT NULL DEFAULT 0,
    mtime     TIMESTAMPTZ,
    sha256    TEXT        NOT NULL DEFAULT '',
    outcome   TEXT        NOT NULL DEFAULT '',
    match_id  BIGINT,
    error     TEXT        NOT NULL DEFAULT '',
    CONSTRAINT import_batch_file_batch_tenant_fkey
        FOREIGN KEY (tenant_id, batch_id) REFERENCES import_batch (tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT import_batch_file_match_tenant_fkey
        FOREIGN KEY (tenant_id, match_id) REFERENCES match (tenant_id, id) ON DELETE SET NULL (match_id)
);
CREATE INDEX IF NOT EXISTS idx_import_batch_file_batch ON import_batch_file (tenant_id, batch_id);
CREATE INDEX IF NOT EXISTS idx_import_batch_file_path  ON import_batch_file (tenant_id, path, size, mtime);

CREATE TABLE IF NOT EXISTS player_alias (
    tenant_id  BIGINT      NOT NULL,
    alias      TEXT        NOT NULL,
    canonical  TEXT        NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (tenant_id, alias)
);
CREATE INDEX IF NOT EXISTS idx_player_alias_canonical ON player_alias (tenant_id, canonical);

CREATE TABLE IF NOT EXISTS event_alias (
    tenant_id  BIGINT      NOT NULL,
    alias      TEXT        NOT NULL,
    canonical  TEXT        NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (tenant_id, alias)
);
CREATE INDEX IF NOT EXISTS idx_event_alias_canonical ON event_alias (tenant_id, canonical);
