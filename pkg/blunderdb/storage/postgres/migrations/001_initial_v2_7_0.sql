-- blunderDB PostgreSQL schema — initial migration.
--
-- This is the terminal SQLite schema (v2.7.0) translated to PostgreSQL. The
-- 15 historical SQLite migrations are deliberately NOT ported: PostgreSQL
-- starts fresh here and tracks its own forward chain (002_*.sql onward).
-- See migrations/README.md.
--
-- Multi-tenancy: every domain table carries `tenant_id BIGINT NOT NULL`.
-- A single shared schema holds all tenants; the application filters by
-- tenant_id on every query. Per-tenant Zobrist dedup is enforced by the
-- composite unique index (tenant_id, zobrist_hash). The `metadata` table is
-- database-level infrastructure (schema version) and has no tenant_id.
--
-- Tables are created in dependency order so inline foreign keys resolve.

CREATE TABLE IF NOT EXISTS position (
    id                BIGSERIAL PRIMARY KEY,
    tenant_id         BIGINT  NOT NULL,
    zobrist_hash      BIGINT,
    decision_type     BIGINT,
    player_on_roll    BIGINT,
    dice_1            BIGINT,
    dice_2            BIGINT,
    cube_value        BIGINT,
    cube_owner        BIGINT,
    score_1           BIGINT,
    score_2           BIGINT,
    match_length      BIGINT,
    has_jacoby        BOOLEAN,
    has_beaver        BOOLEAN,
    pip_1             BIGINT,
    pip_2             BIGINT,
    pip_diff          BIGINT,
    off_1             BIGINT,
    off_2             BIGINT,
    back_checkers_1   BIGINT,
    back_checkers_2   BIGINT,
    no_contact        BOOLEAN,
    occupancy_1       BIGINT,
    occupancy_2       BIGINT,
    point_mask_1      BIGINT,
    point_mask_2      BIGINT,
    state             TEXT    NOT NULL,
    is_cube_response  BOOLEAN NOT NULL DEFAULT FALSE,
    -- Provenance: the position entered the database on its own rather than
    -- inside a match. Sticky — see docs/adr/0001.
    individually_imported BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS analysis (
    id                       BIGSERIAL PRIMARY KEY,
    tenant_id                BIGINT  NOT NULL,
    position_id              BIGINT  REFERENCES position(id) ON DELETE CASCADE,
    data                     BYTEA,
    best_cube_action         TEXT,
    cube_error               BIGINT,
    best_move_equity_error   BIGINT,
    player1_win_rate         BIGINT,
    player1_gammon_rate      BIGINT,
    player1_backgammon_rate  BIGINT,
    player2_win_rate         BIGINT,
    player2_gammon_rate      BIGINT,
    player2_backgammon_rate  BIGINT,
    is_forced                BOOLEAN NOT NULL DEFAULT FALSE,
    is_close_cube            BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS comment (
    id           BIGSERIAL PRIMARY KEY,
    tenant_id    BIGINT NOT NULL,
    position_id  BIGINT REFERENCES position(id) ON DELETE CASCADE,
    text         TEXT,
    created_at   TIMESTAMPTZ DEFAULT now(),
    modified_at  TIMESTAMPTZ
);

-- Database-level infrastructure: schema version, etc. Not tenant-scoped.
CREATE TABLE IF NOT EXISTS metadata (
    key    TEXT PRIMARY KEY,
    value  TEXT
);

CREATE TABLE IF NOT EXISTS command_history (
    id         BIGSERIAL PRIMARY KEY,
    tenant_id  BIGINT NOT NULL,
    command    TEXT,
    timestamp  TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS filter_library (
    id             BIGSERIAL PRIMARY KEY,
    tenant_id      BIGINT NOT NULL,
    name           TEXT,
    command        TEXT,
    edit_position  TEXT
);

CREATE TABLE IF NOT EXISTS search_history (
    id         BIGSERIAL PRIMARY KEY,
    tenant_id  BIGINT NOT NULL,
    command    TEXT,
    position   TEXT,
    timestamp  BIGINT
);

CREATE TABLE IF NOT EXISTS tournament (
    id          BIGSERIAL PRIMARY KEY,
    tenant_id   BIGINT NOT NULL,
    name        TEXT NOT NULL,
    date        TEXT,
    location    TEXT,
    sort_order  BIGINT DEFAULT 0,
    comment     TEXT DEFAULT '',
    created_at  TIMESTAMPTZ DEFAULT now(),
    updated_at  TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS match (
    id                     BIGSERIAL PRIMARY KEY,
    tenant_id              BIGINT NOT NULL,
    player1_name           TEXT,
    player2_name           TEXT,
    event                  TEXT,
    location               TEXT,
    round                  TEXT,
    match_length           BIGINT,
    match_date             TIMESTAMPTZ,
    import_date            TIMESTAMPTZ DEFAULT now(),
    file_path              TEXT,
    game_count             BIGINT DEFAULT 0,
    match_hash             TEXT,
    tournament_id          BIGINT REFERENCES tournament(id) ON DELETE SET NULL,
    last_visited_position  BIGINT DEFAULT -1,
    canonical_hash         TEXT,
    comment                TEXT DEFAULT '',
    tournament_sort_order  BIGINT DEFAULT 0
);

CREATE TABLE IF NOT EXISTS game (
    id               BIGSERIAL PRIMARY KEY,
    tenant_id        BIGINT NOT NULL,
    match_id         BIGINT REFERENCES match(id) ON DELETE CASCADE,
    game_number      BIGINT,
    initial_score_1  BIGINT,
    initial_score_2  BIGINT,
    winner           BIGINT,
    points_won       BIGINT,
    move_count       BIGINT DEFAULT 0
);

CREATE TABLE IF NOT EXISTS move (
    id            BIGSERIAL PRIMARY KEY,
    tenant_id     BIGINT NOT NULL,
    game_id       BIGINT REFERENCES game(id) ON DELETE CASCADE,
    move_number   BIGINT,
    move_type     TEXT,
    position_id   BIGINT REFERENCES position(id) ON DELETE SET NULL,
    player        BIGINT,
    dice_1        BIGINT,
    dice_2        BIGINT,
    checker_move  TEXT,
    cube_action   TEXT
);

CREATE TABLE IF NOT EXISTS move_analysis (
    id                        BIGSERIAL PRIMARY KEY,
    tenant_id                 BIGINT NOT NULL,
    move_id                   BIGINT REFERENCES move(id) ON DELETE CASCADE,
    analysis_type             TEXT,
    depth                     TEXT,
    equity                    BIGINT,
    equity_error              BIGINT,
    win_rate                  BIGINT,
    gammon_rate               BIGINT,
    backgammon_rate           BIGINT,
    opponent_win_rate         BIGINT,
    opponent_gammon_rate      BIGINT,
    opponent_backgammon_rate  BIGINT
);

CREATE TABLE IF NOT EXISTS collection (
    id           BIGSERIAL PRIMARY KEY,
    tenant_id    BIGINT NOT NULL,
    name         TEXT NOT NULL,
    description  TEXT,
    sort_order   BIGINT DEFAULT 0,
    created_at   TIMESTAMPTZ DEFAULT now(),
    updated_at   TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS collection_position (
    id             BIGSERIAL PRIMARY KEY,
    tenant_id      BIGINT NOT NULL,
    collection_id  BIGINT NOT NULL REFERENCES collection(id) ON DELETE CASCADE,
    position_id    BIGINT NOT NULL REFERENCES position(id) ON DELETE CASCADE,
    sort_order     BIGINT DEFAULT 0,
    added_at       TIMESTAMPTZ DEFAULT now(),
    UNIQUE (collection_id, position_id)
);

CREATE TABLE IF NOT EXISTS anki_deck (
    id                 BIGSERIAL PRIMARY KEY,
    tenant_id          BIGINT NOT NULL,
    name               TEXT NOT NULL,
    description        TEXT DEFAULT '',
    source_type        TEXT NOT NULL DEFAULT 'collection',
    source_id          BIGINT DEFAULT 0,
    source_command     TEXT DEFAULT '',
    request_retention  DOUBLE PRECISION DEFAULT 0.9,
    maximum_interval   DOUBLE PRECISION DEFAULT 36500,
    enable_fuzz        BOOLEAN DEFAULT TRUE,
    created_at         TIMESTAMPTZ DEFAULT now(),
    updated_at         TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS anki_card (
    id              BIGSERIAL PRIMARY KEY,
    tenant_id       BIGINT NOT NULL,
    deck_id         BIGINT NOT NULL REFERENCES anki_deck(id) ON DELETE CASCADE,
    position_id     BIGINT NOT NULL REFERENCES position(id) ON DELETE CASCADE,
    due             TIMESTAMPTZ DEFAULT now(),
    stability       DOUBLE PRECISION DEFAULT 0,
    difficulty      DOUBLE PRECISION DEFAULT 0,
    elapsed_days    BIGINT DEFAULT 0,
    scheduled_days  BIGINT DEFAULT 0,
    reps            BIGINT DEFAULT 0,
    lapses          BIGINT DEFAULT 0,
    state           BIGINT DEFAULT 0,
    last_review     TIMESTAMPTZ,
    suspended       BOOLEAN NOT NULL DEFAULT FALSE,
    buried_until    TIMESTAMPTZ,
    UNIQUE (deck_id, position_id)
);

CREATE TABLE IF NOT EXISTS anki_review_log (
    id              BIGSERIAL PRIMARY KEY,
    tenant_id       BIGINT NOT NULL,
    card_id         BIGINT NOT NULL REFERENCES anki_card(id) ON DELETE CASCADE,
    deck_id         BIGINT NOT NULL,
    position_id     BIGINT NOT NULL,
    rating          BIGINT NOT NULL,
    state           BIGINT NOT NULL DEFAULT 0,
    stability       DOUBLE PRECISION DEFAULT 0,
    difficulty      DOUBLE PRECISION DEFAULT 0,
    elapsed_days    BIGINT DEFAULT 0,
    scheduled_days  BIGINT DEFAULT 0,
    reviewed_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes. Multi-tenant filter columns lead every composite index so the
-- planner can satisfy the always-present `WHERE tenant_id = $1` predicate.
CREATE UNIQUE INDEX IF NOT EXISTS idx_position_zobrist        ON position (tenant_id, zobrist_hash);
CREATE        INDEX IF NOT EXISTS idx_position_decision_pip   ON position (tenant_id, decision_type, pip_diff);
CREATE        INDEX IF NOT EXISTS idx_position_decision_dice  ON position (tenant_id, decision_type, dice_1, dice_2);
CREATE        INDEX IF NOT EXISTS idx_position_cube_response  ON position (tenant_id, decision_type) WHERE is_cube_response;
CREATE        INDEX IF NOT EXISTS idx_position_individual      ON position (tenant_id) WHERE individually_imported;
CREATE        INDEX IF NOT EXISTS idx_position_pip_diff       ON position (tenant_id, pip_diff);
CREATE        INDEX IF NOT EXISTS idx_position_dice           ON position (tenant_id, dice_1, dice_2);
CREATE        INDEX IF NOT EXISTS idx_position_off            ON position (tenant_id, off_1, off_2);
CREATE        INDEX IF NOT EXISTS idx_position_score          ON position (tenant_id, match_length, score_1, score_2);
CREATE        INDEX IF NOT EXISTS idx_position_score_cube     ON position (tenant_id, match_length, score_1, score_2, cube_value);
CREATE        INDEX IF NOT EXISTS idx_analysis_position       ON analysis (position_id);
CREATE        INDEX IF NOT EXISTS idx_analysis_win_gammon     ON analysis (tenant_id, player1_win_rate, player1_gammon_rate);
CREATE        INDEX IF NOT EXISTS idx_analysis_win1           ON analysis (tenant_id, player1_win_rate);
CREATE        INDEX IF NOT EXISTS idx_analysis_cube_error     ON analysis (tenant_id, cube_error);
CREATE        INDEX IF NOT EXISTS idx_analysis_move_error     ON analysis (tenant_id, best_move_equity_error);
CREATE        INDEX IF NOT EXISTS idx_analysis_is_forced      ON analysis (tenant_id) WHERE is_forced;
CREATE        INDEX IF NOT EXISTS idx_analysis_is_close_cube  ON analysis (tenant_id) WHERE is_close_cube;
CREATE        INDEX IF NOT EXISTS idx_match_hash              ON match (tenant_id, match_hash);
CREATE UNIQUE INDEX IF NOT EXISTS idx_match_canonical         ON match (tenant_id, canonical_hash);
CREATE        INDEX IF NOT EXISTS idx_move_position           ON move (position_id);
CREATE        INDEX IF NOT EXISTS idx_move_game               ON move (game_id);
CREATE        INDEX IF NOT EXISTS idx_game_match              ON game (match_id);
CREATE        INDEX IF NOT EXISTS idx_collection_position_collection ON collection_position (collection_id);
CREATE        INDEX IF NOT EXISTS idx_anki_card_deck          ON anki_card (deck_id);
CREATE        INDEX IF NOT EXISTS idx_anki_card_due           ON anki_card (deck_id, due);
CREATE        INDEX IF NOT EXISTS idx_anki_review_log_card    ON anki_review_log (tenant_id, card_id, reviewed_at);
CREATE        INDEX IF NOT EXISTS idx_anki_review_log_deck    ON anki_review_log (tenant_id, deck_id, reviewed_at);
