-- Forward migration: a board filter that places all fifteen checkers of a side implies its
-- whole occupancy, an equality an index answers (no schema change, DatabaseVersion untouched).
--
-- idx_move_game_position: the positions of a game read from the index alone, tenant_id
-- included because row-level security reads it (supersedes idx_move_game, a prefix of it).
--
-- idx_analysis_blunder: from the threshold of a blunder up, a search on the played-move or
-- cube error reads a list in id order, a few percent of the tenant's rows.
--
-- Back: DROP these four, then CREATE INDEX idx_move_game ON move (game_id);

CREATE INDEX IF NOT EXISTS idx_position_occupancy1 ON position (tenant_id, occupancy_1);
CREATE INDEX IF NOT EXISTS idx_position_occupancy2 ON position (tenant_id, occupancy_2);
CREATE INDEX IF NOT EXISTS idx_move_game_position ON move (game_id, position_id) INCLUDE (tenant_id);
DROP INDEX IF EXISTS idx_move_game;
CREATE INDEX IF NOT EXISTS idx_analysis_blunder ON analysis (tenant_id, position_id) WHERE best_move_equity_error >= 100 OR cube_error >= 100;
