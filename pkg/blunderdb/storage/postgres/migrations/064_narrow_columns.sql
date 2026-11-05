-- Forward migration: columns as wide as their values, 19 GB out of 137 on the
-- largest library (no schema change a reader sees, DatabaseVersion untouched).
--
-- 16 bits for what the board bounds (dice, pips, off, back checkers, rates in
-- hundredths of a percent, codes); 32 bits for what a caller can write freely
-- (cube, scores, match length), for the masks (26 bits) and for errors in
-- millipoints. Ids, tenant, hash and the times of a move stay 64 bits. A value
-- out of range is refused by name (errors_postgres.go), never cut; a search
-- compares its integers as bigint (adapter.go), so a filter past the bound finds
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

-- A value past its type fails the ALTER (22003) without saying where: the
-- refusal names the table, the column, the value and the row, which the
-- operator corrects or deletes before starting again. The search lifts FORCE,
-- the migrating connection carries no tenant; the refusal rolls it back.
CREATE OR REPLACE FUNCTION blunderdb_narrow_table(tbl text, probe text, cols text[]) RETURNS void
LANGUAGE plpgsql AS $$
DECLARE
    alters text;
    bad text;
    row_id bigint;
BEGIN
    IF (SELECT atttypid <> 'bigint'::regtype FROM pg_attribute
         WHERE attrelid = tbl::regclass AND attname = probe) THEN
        RETURN;
    END IF;
    SELECT string_agg(format('ALTER COLUMN %I TYPE %s', cols[g][1], cols[g][2]), ', ')
      INTO alters FROM generate_subscripts(cols, 1) AS g;
    BEGIN
        EXECUTE format('ALTER TABLE %I %s', tbl, alters);
    EXCEPTION WHEN numeric_value_out_of_range THEN
        EXECUTE format('ALTER TABLE %I NO FORCE ROW LEVEL SECURITY', tbl);
        FOR c IN 1 .. array_length(cols, 1) LOOP
            EXECUTE format('SELECT CAST(%1$I AS text), id FROM %2$I WHERE %1$I NOT BETWEEN $1 AND $2 LIMIT 1', cols[c][1], tbl)
               INTO bad, row_id
              USING CASE cols[c][2] WHEN 'smallint' THEN -32768 ELSE -2147483648 END,
                    CASE cols[c][2] WHEN 'smallint' THEN 32767 ELSE 2147483647 END;
            IF bad IS NOT NULL THEN
                RAISE EXCEPTION 'blunderdb: %.% holds %, past % (row id %): correct or delete that row, then start again',
                    tbl, cols[c][1], bad, cols[c][2], row_id USING ERRCODE = 'numeric_value_out_of_range';
            END IF;
        END LOOP;
        RAISE;
    END;
END $$;

CREATE OR REPLACE FUNCTION blunderdb_narrow_columns() RETURNS void
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM blunderdb_narrow_table('position', 'pip_1', ARRAY[
        ['decision_type', 'smallint'],
        ['player_on_roll', 'smallint'],
        ['dice_1', 'smallint'],
        ['dice_2', 'smallint'],
        ['cube_value', 'integer'],
        ['cube_owner', 'smallint'],
        ['score_1', 'integer'],
        ['score_2', 'integer'],
        ['match_length', 'integer'],
        ['pip_1', 'smallint'],
        ['pip_2', 'smallint'],
        ['pip_diff', 'smallint'],
        ['off_1', 'smallint'],
        ['off_2', 'smallint'],
        ['back_checkers_1', 'smallint'],
        ['back_checkers_2', 'smallint'],
        ['occupancy_1', 'integer'],
        ['occupancy_2', 'integer'],
        ['point_mask_1', 'integer'],
        ['point_mask_2', 'integer'],
        ['game_phase', 'smallint'],
        ['max_cube', 'smallint'],
        ['game_type', 'smallint']]);
    PERFORM blunderdb_narrow_table('analysis', 'player1_win_rate', ARRAY[
        ['cube_error', 'integer'],
        ['best_move_equity_error', 'integer'],
        ['player1_win_rate', 'smallint'],
        ['player1_gammon_rate', 'smallint'],
        ['player1_backgammon_rate', 'smallint'],
        ['player2_win_rate', 'smallint'],
        ['player2_gammon_rate', 'smallint'],
        ['player2_backgammon_rate', 'smallint'],
        ['analysis_depth', 'smallint'],
        ['best_cube_action', 'smallint']]);
    PERFORM blunderdb_narrow_table('move', 'move_number', ARRAY[
        ['move_number', 'integer'],
        ['player', 'smallint'],
        ['dice_1', 'smallint'],
        ['dice_2', 'smallint'],
        ['luck_mp', 'integer'],
        ['error_mp', 'integer'],
        ['move_type', 'smallint'],
        ['cube_action', 'smallint'],
        ['decision_error_mp', 'integer'],
        ['is_close_cube', 'smallint']]);
END $$;

DO $$ BEGIN
    IF pg_relation_size('position') <= 536870912 THEN
        PERFORM blunderdb_narrow_columns();
    ELSE
        RAISE WARNING 'position is too large to rewrite at a start: columns stay wide, run SELECT blunderdb_narrow_columns() in a window';
    END IF;
END $$;
