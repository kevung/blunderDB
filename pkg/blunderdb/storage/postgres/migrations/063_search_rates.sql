-- Forward migration: the win rates take a plain index, like the gammon rates in 061 (no schema
-- change, DatabaseVersion untouched).
--
-- idx_analysis_win1/2 replace the two covering indexes of 008 and 032: position_id and the gammon
-- rate in the key weighed five times the plain index, for no gain a search can read (figures in
-- docs/PERF-RECHERCHE.md, gammonGo). SQLite keeps its covering indexes: not measured there.
--
-- Back: DROP these two, then
--   CREATE INDEX idx_analysis_win_gammon_covering ON analysis (tenant_id, player1_win_rate, player1_gammon_rate, position_id);
--   CREATE INDEX idx_analysis_win_gammon2_covering ON analysis (tenant_id, player2_win_rate, player2_gammon_rate, position_id);

-- An index build holds SHARE on the table until this file's transaction ends: lock_timeout makes
-- a busy table fail the migration, retried at the next start, rather than queue writers behind it.
SET LOCAL lock_timeout = '15s';

CREATE INDEX IF NOT EXISTS idx_analysis_win1 ON analysis (tenant_id, player1_win_rate);
CREATE INDEX IF NOT EXISTS idx_analysis_win2 ON analysis (tenant_id, player2_win_rate);
DROP INDEX IF EXISTS idx_analysis_win_gammon_covering;
DROP INDEX IF EXISTS idx_analysis_win_gammon2_covering;
