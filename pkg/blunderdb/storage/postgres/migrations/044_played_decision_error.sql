-- Forward migration: the 2.39.0 wave — a decision's error is the error of the
-- move played in that match.
--
--   * move.decision_error_mp, move.is_close_cube — the analysis's
--     cube_error / best_move_equity_error and is_close_cube scored against
--     the move's own play (sqlshared/played_decisions.go). The analysis row
--     scores one play per position, and a position reached in two matches
--     with two plays charged both with the first one's error.
--
-- Both are projections of the compressed analysis blob, which SQL cannot
-- read: Migrate's Go-side passes (runGoBackfills, generation 2) score them,
-- recompute the 2.37.0 columns 042 left to `blunderdb repair`, and drop
-- match_stats — the rows 042 and 043 meant to drop too, which FORCEd RLS hid
-- from them.
-- Schema-visible: bumps domain.DatabaseVersion to 2.39.0.

ALTER TABLE move ADD COLUMN IF NOT EXISTS decision_error_mp BIGINT;
ALTER TABLE move ADD COLUMN IF NOT EXISTS is_close_cube     INTEGER NOT NULL DEFAULT 0;
