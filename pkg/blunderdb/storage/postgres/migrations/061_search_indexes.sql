-- Forward migration: indexes of the search, measured on five million positions
-- (no schema change, DatabaseVersion untouched).
--
-- idx_analysis_tenant_position: idx_analysis_position mixes every tenant, so a
-- walk in id order paid for the rows of the others. idx_analysis_error_order
-- orders the search by played-move error, and supersedes idx_analysis_move_error,
-- a prefix of it. idx_analysis_gammon1/2: the covering index leads with the win
-- rate and cannot answer a gammon rate alone. idx_position_structure: the checker
-- masks of a tenant in id order, so that a structure is read without the heap.
--
-- Back: DROP these five, then
--   CREATE INDEX idx_analysis_move_error ON analysis (tenant_id, best_move_equity_error);

CREATE INDEX IF NOT EXISTS idx_analysis_tenant_position ON analysis (tenant_id, position_id);
CREATE INDEX IF NOT EXISTS idx_analysis_error_order ON analysis (tenant_id, best_move_equity_error DESC, position_id);
DROP INDEX IF EXISTS idx_analysis_move_error;
CREATE INDEX IF NOT EXISTS idx_analysis_gammon1 ON analysis (tenant_id, player1_gammon_rate);
CREATE INDEX IF NOT EXISTS idx_analysis_gammon2 ON analysis (tenant_id, player2_gammon_rate);
CREATE INDEX IF NOT EXISTS idx_position_structure ON position (tenant_id, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2);
