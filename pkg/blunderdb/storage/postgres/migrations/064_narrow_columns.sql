-- Forward migration: columns as wide as their values, 19 GB out of 137 on the
-- largest library (no schema change a reader sees, DatabaseVersion untouched).
--
-- 16 bits for what the board bounds (dice, pips, off, back checkers, rates in
-- hundredths of a percent, codes); 32 bits for what a caller can write freely
-- (cube, scores, match length), for the masks (26 bits) and for errors in
-- millipoints. Ids, tenant, hash and the times of a move stay 64 bits. A value
-- out of range is refused by name (errors_postgres.go), never cut; a search
-- binds its integers as bigint (adapter.go), so a filter past the bound finds
-- nothing rather than fails.
--
-- One ALTER per table: one rewrite, under ACCESS EXCLUSIVE, about 12 000 rows/s
-- with its indexes. Past 512 MB of position table (1,5 M rows, two minutes) that is longer of closed
-- library, which no start decides alone: the file records itself and leaves
-- the columns wide, and everything still works on wide columns. Then, in a
-- window: SELECT blunderdb_narrow_columns();
-- Back: the same ALTERs TYPE bigint. Idempotent: a narrowed table is skipped.
-- No COMMIT, no role switch: it holds on blunderdb_user under FORCE RLS.

SET LOCAL lock_timeout = '15s';

CREATE OR REPLACE FUNCTION blunderdb_narrow_columns() RETURNS void
LANGUAGE plpgsql AS $$
BEGIN
    IF (SELECT atttypid = 'bigint'::regtype FROM pg_attribute
         WHERE attrelid = 'position'::regclass AND attname = 'pip_1') THEN
        ALTER TABLE position
            ALTER COLUMN decision_type TYPE smallint,
            ALTER COLUMN player_on_roll TYPE smallint,
            ALTER COLUMN dice_1 TYPE smallint,
            ALTER COLUMN dice_2 TYPE smallint,
            ALTER COLUMN cube_value TYPE integer,
            ALTER COLUMN cube_owner TYPE smallint,
            ALTER COLUMN score_1 TYPE integer,
            ALTER COLUMN score_2 TYPE integer,
            ALTER COLUMN match_length TYPE integer,
            ALTER COLUMN pip_1 TYPE smallint,
            ALTER COLUMN pip_2 TYPE smallint,
            ALTER COLUMN pip_diff TYPE smallint,
            ALTER COLUMN off_1 TYPE smallint,
            ALTER COLUMN off_2 TYPE smallint,
            ALTER COLUMN back_checkers_1 TYPE smallint,
            ALTER COLUMN back_checkers_2 TYPE smallint,
            ALTER COLUMN occupancy_1 TYPE integer,
            ALTER COLUMN occupancy_2 TYPE integer,
            ALTER COLUMN point_mask_1 TYPE integer,
            ALTER COLUMN point_mask_2 TYPE integer,
            ALTER COLUMN game_phase TYPE smallint,
            ALTER COLUMN max_cube TYPE smallint,
            ALTER COLUMN game_type TYPE smallint;
    END IF;
    IF (SELECT atttypid = 'bigint'::regtype FROM pg_attribute
         WHERE attrelid = 'analysis'::regclass AND attname = 'player1_win_rate') THEN
        ALTER TABLE analysis
            ALTER COLUMN cube_error TYPE integer,
            ALTER COLUMN best_move_equity_error TYPE integer,
            ALTER COLUMN player1_win_rate TYPE smallint,
            ALTER COLUMN player1_gammon_rate TYPE smallint,
            ALTER COLUMN player1_backgammon_rate TYPE smallint,
            ALTER COLUMN player2_win_rate TYPE smallint,
            ALTER COLUMN player2_gammon_rate TYPE smallint,
            ALTER COLUMN player2_backgammon_rate TYPE smallint,
            ALTER COLUMN analysis_depth TYPE smallint,
            ALTER COLUMN best_cube_action TYPE smallint;
    END IF;
    IF (SELECT atttypid = 'bigint'::regtype FROM pg_attribute
         WHERE attrelid = 'move'::regclass AND attname = 'move_number') THEN
        ALTER TABLE move
            ALTER COLUMN move_number TYPE integer,
            ALTER COLUMN player TYPE smallint,
            ALTER COLUMN dice_1 TYPE smallint,
            ALTER COLUMN dice_2 TYPE smallint,
            ALTER COLUMN luck_mp TYPE integer,
            ALTER COLUMN error_mp TYPE integer,
            ALTER COLUMN move_type TYPE smallint,
            ALTER COLUMN cube_action TYPE smallint,
            ALTER COLUMN decision_error_mp TYPE integer,
            ALTER COLUMN is_close_cube TYPE smallint;
    END IF;
END $$;

DO $$ BEGIN
    IF pg_relation_size('position') <= 536870912 THEN
        PERFORM blunderdb_narrow_columns();
    ELSE
        RAISE WARNING 'position is too large to rewrite at a start: columns stay wide, run SELECT blunderdb_narrow_columns() in a window';
    END IF;
END $$;
