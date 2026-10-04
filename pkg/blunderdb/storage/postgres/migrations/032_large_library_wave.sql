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
--   * match.dice_hash and the source metadata of a match (ratings,
--     experience, transcriber, Jacoby/Beaver, engine version).
--   * match_stats — per-match, per-seat tallies the corpus statistics read
--     (schema_sqlite.go states the arithmetic); filled by the stats store.
--   * training_item.position_id / answer / error_mp and comment.author.
--   * Index pruning, decided on the query plans of a 15.6 M-position library
--     (tasks/search-query-plans.txt): no index leads with decision_type, the
--     cube-response index keeps only its selective side, and the player-2
--     rates get the covering index the player-1 rates have.
-- Schema-visible: bumps domain.DatabaseVersion to 2.30.0.

ALTER TABLE analysis ADD COLUMN IF NOT EXISTS analysis_engine TEXT;
ALTER TABLE analysis ADD COLUMN IF NOT EXISTS analysis_depth  INTEGER;
ALTER TABLE analysis ADD COLUMN IF NOT EXISTS creation_date   TIMESTAMPTZ;
ALTER TABLE position ADD COLUMN IF NOT EXISTS match_date      TIMESTAMPTZ;

-- Row-Level Security, once applied, is FORCEd on the tables the backfill
-- reads and writes: the migrating connection carries no tenant and would see
-- no row, so the UPDATE would date nothing and say nothing. FORCE is lifted
-- for the statement and put back; the batch runs as one transaction.
CREATE TEMP TABLE match_date_forced (name TEXT) ON COMMIT DROP;

DO $$
DECLARE t TEXT;
BEGIN
    FOREACH t IN ARRAY ARRAY['position', 'move', 'game', 'match'] LOOP
        IF EXISTS (SELECT 1 FROM pg_class WHERE oid = to_regclass(t) AND relforcerowsecurity) THEN
            EXECUTE format('ALTER TABLE %I NO FORCE ROW LEVEL SECURITY', t);
            INSERT INTO match_date_forced VALUES (t);
        END IF;
    END LOOP;
END $$;

UPDATE position p SET match_date = t.md
FROM (SELECT mv.tenant_id, mv.position_id, MIN(m.match_date) AS md
        FROM move mv
        JOIN game g  ON g.id = mv.game_id AND g.tenant_id = mv.tenant_id
        JOIN match m ON m.id = g.match_id AND m.tenant_id = g.tenant_id
       WHERE mv.position_id IS NOT NULL
       GROUP BY mv.tenant_id, mv.position_id) t
WHERE p.tenant_id = t.tenant_id AND p.id = t.position_id AND p.match_date IS NULL;

DO $$
DECLARE t TEXT;
BEGIN
    FOR t IN SELECT name FROM match_date_forced LOOP
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
    END LOOP;
END $$;

DROP INDEX IF EXISTS idx_position_decision_dice;
DROP INDEX IF EXISTS idx_position_decision_pip;
DROP INDEX IF EXISTS idx_position_cube_response;
DROP INDEX IF EXISTS idx_analysis_win2;
DROP INDEX IF EXISTS idx_analysis_gammon2;
DROP INDEX IF EXISTS idx_position_game_phase;

CREATE INDEX IF NOT EXISTS idx_position_phase_off ON position (tenant_id, game_phase, off_1);
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

ALTER TABLE match ADD COLUMN IF NOT EXISTS dice_hash          TEXT;
ALTER TABLE match ADD COLUMN IF NOT EXISTS player1_elo        DOUBLE PRECISION;
ALTER TABLE match ADD COLUMN IF NOT EXISTS player2_elo        DOUBLE PRECISION;
ALTER TABLE match ADD COLUMN IF NOT EXISTS player1_experience INTEGER;
ALTER TABLE match ADD COLUMN IF NOT EXISTS player2_experience INTEGER;
ALTER TABLE match ADD COLUMN IF NOT EXISTS transcriber        TEXT DEFAULT '';
ALTER TABLE match ADD COLUMN IF NOT EXISTS has_jacoby         BOOLEAN;
ALTER TABLE match ADD COLUMN IF NOT EXISTS has_beaver         BOOLEAN;
ALTER TABLE match ADD COLUMN IF NOT EXISTS engine_version     TEXT DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_match_dice_hash ON match (tenant_id, dice_hash) WHERE dice_hash IS NOT NULL;

CREATE TABLE IF NOT EXISTS match_stats (
    tenant_id         BIGINT  NOT NULL,
    match_id          BIGINT  NOT NULL,
    seat              INTEGER NOT NULL CHECK (seat IN (1, 2)),
    decisions         INTEGER NOT NULL DEFAULT 0,
    checker_decisions INTEGER NOT NULL DEFAULT 0,
    cube_decisions    INTEGER NOT NULL DEFAULT 0,
    error_mp          BIGINT  NOT NULL DEFAULT 0,
    pr                DOUBLE PRECISION,
    luck_mp           BIGINT  NOT NULL DEFAULT 0,
    luck_rolls        INTEGER NOT NULL DEFAULT 0,
    blunders          INTEGER NOT NULL DEFAULT 0,
    analysis_engine   TEXT,
    analysis_depth    INTEGER,
    computed_at       TIMESTAMPTZ DEFAULT now(),
    checker_error_mp  BIGINT,
    cube_error_mp     BIGINT,
    errors            INTEGER,
    snowie_error_mp   BIGINT,
    snowie_moves      INTEGER,
    checker_moves     INTEGER,
    PRIMARY KEY (tenant_id, match_id, seat),
    CONSTRAINT match_stats_match_tenant_fkey
        FOREIGN KEY (tenant_id, match_id) REFERENCES match (tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_match_stats_pr ON match_stats (tenant_id, pr) WHERE pr IS NOT NULL;

ALTER TABLE training_item ADD COLUMN IF NOT EXISTS position_id BIGINT;
ALTER TABLE training_item ADD COLUMN IF NOT EXISTS answer      TEXT NOT NULL DEFAULT '';
ALTER TABLE training_item ADD COLUMN IF NOT EXISTS error_mp    INTEGER;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'training_item_position_fkey') THEN
        ALTER TABLE training_item ADD CONSTRAINT training_item_position_fkey
            FOREIGN KEY (tenant_id, position_id)
            REFERENCES position (tenant_id, id) ON DELETE SET NULL (position_id);
    END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_training_item_position ON training_item (tenant_id, position_id) WHERE position_id IS NOT NULL;

ALTER TABLE comment ADD COLUMN IF NOT EXISTS author TEXT NOT NULL DEFAULT '';
