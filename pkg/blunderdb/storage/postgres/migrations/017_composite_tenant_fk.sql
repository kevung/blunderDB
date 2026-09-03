-- Forward migration: every foreign key between two tenant-scoped tables
-- becomes a composite (tenant_id, parent_id) REFERENCES parent (tenant_id,
-- id), instead of parent_id REFERENCES parent (id) alone (#235).
--
-- id is already globally unique (BIGSERIAL), so the single-column form was
-- never wrong about WHICH row a foreign key points to — only silent about
-- whether that row belongs to the same tenant. A child row's tenant_id
-- naming one tenant while its parent belonged to another was a mistake only
-- the application layer's own filtering prevented; the database itself
-- would happily store it. The composite form makes that structurally
-- impossible from here on.
--
-- Constraint-only, like 006/008/010: no column, no table, no data changes,
-- nothing domain-visible, so domain.DatabaseVersion is left alone (see
-- migrations/README.md's "index-only migration" exception, which this
-- follows even though it is neither an index nor read-only).
--
-- Every table another table references gets `UNIQUE (tenant_id, id)` first
-- (needed as a composite foreign key's target); the old single-column
-- foreign keys are then dropped by their default Postgres-assigned name
-- (`<table>_<column>_fkey`) and replaced by the named composite ones —
-- exactly the shape 001_initial_v2_7_0.sql now creates directly, so this
-- migration is a no-op (every guard below finds its target already in
-- place) on a database that was bootstrapped after this migration existed.
--
-- If ADD CONSTRAINT below ever fails on an existing installation, some row
-- already has a tenant_id that disagrees with its parent's — a symptom of
-- the exact bug this migration closes off, not a reason to skip it. Find it
-- first, e.g. for analysis/position:
--   SELECT a.id FROM analysis a JOIN position p ON p.id = a.position_id
--   WHERE a.tenant_id <> p.tenant_id;

-- --- composite targets: UNIQUE (tenant_id, id) on every referenced table ---
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'position_tenant_id_key') THEN
        ALTER TABLE position ADD CONSTRAINT position_tenant_id_key UNIQUE (tenant_id, id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'tournament_tenant_id_key') THEN
        ALTER TABLE tournament ADD CONSTRAINT tournament_tenant_id_key UNIQUE (tenant_id, id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'match_tenant_id_key') THEN
        ALTER TABLE match ADD CONSTRAINT match_tenant_id_key UNIQUE (tenant_id, id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'game_tenant_id_key') THEN
        ALTER TABLE game ADD CONSTRAINT game_tenant_id_key UNIQUE (tenant_id, id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'move_tenant_id_key') THEN
        ALTER TABLE move ADD CONSTRAINT move_tenant_id_key UNIQUE (tenant_id, id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'collection_tenant_id_key') THEN
        ALTER TABLE collection ADD CONSTRAINT collection_tenant_id_key UNIQUE (tenant_id, id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'anki_deck_tenant_id_key') THEN
        ALTER TABLE anki_deck ADD CONSTRAINT anki_deck_tenant_id_key UNIQUE (tenant_id, id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'anki_card_tenant_id_key') THEN
        ALTER TABLE anki_card ADD CONSTRAINT anki_card_tenant_id_key UNIQUE (tenant_id, id);
    END IF;
END $$;

-- --- analysis.position_id ---
ALTER TABLE analysis DROP CONSTRAINT IF EXISTS analysis_position_id_fkey;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'analysis_position_tenant_fkey') THEN
        ALTER TABLE analysis ADD CONSTRAINT analysis_position_tenant_fkey
            FOREIGN KEY (tenant_id, position_id) REFERENCES position (tenant_id, id) ON DELETE CASCADE;
    END IF;
END $$;

-- --- comment.position_id ---
ALTER TABLE comment DROP CONSTRAINT IF EXISTS comment_position_id_fkey;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'comment_position_tenant_fkey') THEN
        ALTER TABLE comment ADD CONSTRAINT comment_position_tenant_fkey
            FOREIGN KEY (tenant_id, position_id) REFERENCES position (tenant_id, id) ON DELETE CASCADE;
    END IF;
END $$;

-- --- match.tournament_id (SET NULL names only tournament_id: tenant_id must
-- never be nulled out by a tournament's deletion) ---
ALTER TABLE match DROP CONSTRAINT IF EXISTS match_tournament_id_fkey;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'match_tournament_tenant_fkey') THEN
        ALTER TABLE match ADD CONSTRAINT match_tournament_tenant_fkey
            FOREIGN KEY (tenant_id, tournament_id) REFERENCES tournament (tenant_id, id) ON DELETE SET NULL (tournament_id);
    END IF;
END $$;

-- --- game.match_id ---
ALTER TABLE game DROP CONSTRAINT IF EXISTS game_match_id_fkey;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'game_match_tenant_fkey') THEN
        ALTER TABLE game ADD CONSTRAINT game_match_tenant_fkey
            FOREIGN KEY (tenant_id, match_id) REFERENCES match (tenant_id, id) ON DELETE CASCADE;
    END IF;
END $$;

-- --- move.game_id, move.position_id (SET NULL names only position_id) ---
ALTER TABLE move DROP CONSTRAINT IF EXISTS move_game_id_fkey;
ALTER TABLE move DROP CONSTRAINT IF EXISTS move_position_id_fkey;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'move_game_tenant_fkey') THEN
        ALTER TABLE move ADD CONSTRAINT move_game_tenant_fkey
            FOREIGN KEY (tenant_id, game_id) REFERENCES game (tenant_id, id) ON DELETE CASCADE;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'move_position_tenant_fkey') THEN
        ALTER TABLE move ADD CONSTRAINT move_position_tenant_fkey
            FOREIGN KEY (tenant_id, position_id) REFERENCES position (tenant_id, id) ON DELETE SET NULL (position_id);
    END IF;
END $$;

-- --- move_analysis.move_id ---
ALTER TABLE move_analysis DROP CONSTRAINT IF EXISTS move_analysis_move_id_fkey;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'move_analysis_move_tenant_fkey') THEN
        ALTER TABLE move_analysis ADD CONSTRAINT move_analysis_move_tenant_fkey
            FOREIGN KEY (tenant_id, move_id) REFERENCES move (tenant_id, id) ON DELETE CASCADE;
    END IF;
END $$;

-- --- collection_position.collection_id, collection_position.position_id ---
ALTER TABLE collection_position DROP CONSTRAINT IF EXISTS collection_position_collection_id_fkey;
ALTER TABLE collection_position DROP CONSTRAINT IF EXISTS collection_position_position_id_fkey;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'collection_position_collection_tenant_fkey') THEN
        ALTER TABLE collection_position ADD CONSTRAINT collection_position_collection_tenant_fkey
            FOREIGN KEY (tenant_id, collection_id) REFERENCES collection (tenant_id, id) ON DELETE CASCADE;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'collection_position_position_tenant_fkey') THEN
        ALTER TABLE collection_position ADD CONSTRAINT collection_position_position_tenant_fkey
            FOREIGN KEY (tenant_id, position_id) REFERENCES position (tenant_id, id) ON DELETE CASCADE;
    END IF;
END $$;

-- --- anki_card.deck_id, anki_card.position_id ---
ALTER TABLE anki_card DROP CONSTRAINT IF EXISTS anki_card_deck_id_fkey;
ALTER TABLE anki_card DROP CONSTRAINT IF EXISTS anki_card_position_id_fkey;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'anki_card_deck_tenant_fkey') THEN
        ALTER TABLE anki_card ADD CONSTRAINT anki_card_deck_tenant_fkey
            FOREIGN KEY (tenant_id, deck_id) REFERENCES anki_deck (tenant_id, id) ON DELETE CASCADE;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'anki_card_position_tenant_fkey') THEN
        ALTER TABLE anki_card ADD CONSTRAINT anki_card_position_tenant_fkey
            FOREIGN KEY (tenant_id, position_id) REFERENCES position (tenant_id, id) ON DELETE CASCADE;
    END IF;
END $$;

-- --- anki_review_log.card_id ---
ALTER TABLE anki_review_log DROP CONSTRAINT IF EXISTS anki_review_log_card_id_fkey;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'anki_review_log_card_tenant_fkey') THEN
        ALTER TABLE anki_review_log ADD CONSTRAINT anki_review_log_card_tenant_fkey
            FOREIGN KEY (tenant_id, card_id) REFERENCES anki_card (tenant_id, id) ON DELETE CASCADE;
    END IF;
END $$;

-- --- anki_review_log.deck_id, anki_review_log.position_id ---
-- 016 (issue #185) already adds these as composite (tenant_id, …), NOT VALID
-- foreign keys directly, so this is a no-op there; it only matters for a
-- database that ran 016's original single-column form
-- (anki_review_log_deck_fk / anki_review_log_position_fk) before this
-- migration existed. Kept NOT VALID like 016's own: the orphan risk 016
-- documents (issue #157) is unchanged by adding tenant_id to the key.
ALTER TABLE anki_review_log DROP CONSTRAINT IF EXISTS anki_review_log_deck_fk;
ALTER TABLE anki_review_log DROP CONSTRAINT IF EXISTS anki_review_log_position_fk;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'anki_review_log_deck_tenant_fkey') THEN
        ALTER TABLE anki_review_log ADD CONSTRAINT anki_review_log_deck_tenant_fkey
            FOREIGN KEY (tenant_id, deck_id) REFERENCES anki_deck (tenant_id, id) ON DELETE CASCADE NOT VALID;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'anki_review_log_position_tenant_fkey') THEN
        ALTER TABLE anki_review_log ADD CONSTRAINT anki_review_log_position_tenant_fkey
            FOREIGN KEY (tenant_id, position_id) REFERENCES position (tenant_id, id) ON DELETE CASCADE NOT VALID;
    END IF;
END $$;
