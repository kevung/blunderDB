-- Forward migration: the 2.48.0 wave — match_stats keeps each seat's MWC
-- loss and its per-game spread, so the match and tournament badges and the
-- players table's MWC7 read the table instead of the decisions.
--
--   * mwc_loss     — the seat's MWC loss over its counted decisions; NULL
--                    when none has a value at its score.
--   * mwc_games    — the games either seat's counted decisions reach; NULL
--                    only on a row written before this wave.
--   * mwc_game_ss  — the sum of squared deviations of the seat's per-game
--                    losses (domain.MatchMWC7Spread).
--
-- The rows written before carry NULL in mwc_games: the older-shape repair
-- (sqlshared.OlderShapeMatchStatsPredicate) drops them for every tenant and
-- the next reader recomputes them. A DELETE here would run without a tenant
-- and, under FORCEd row-level security, see no row.
-- Schema-visible: bumps domain.DatabaseVersion to 2.48.0.

ALTER TABLE match_stats ADD COLUMN IF NOT EXISTS mwc_loss DOUBLE PRECISION;
ALTER TABLE match_stats ADD COLUMN IF NOT EXISTS mwc_games INTEGER;
ALTER TABLE match_stats ADD COLUMN IF NOT EXISTS mwc_game_ss DOUBLE PRECISION;
