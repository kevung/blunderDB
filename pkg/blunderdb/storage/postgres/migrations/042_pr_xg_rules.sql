-- Forward migration: the 2.37.0 wave — decisions counted as XG counts them
-- for the Performance Rating.
--
--   * analysis.is_forced — a play is forced when it is the only legal one, or
--     when the analysed candidates cover every legal play at one equity
--     (engine.IsForcedChecker), no longer when one candidate is stored.
--   * analysis.is_close_cube — a no-double counts when its equity is positive
--     and within 0.200 of min(double/take, double/pass)
--     (engine.ComputeIsCloseCube); every double, take and pass counts.
--   * analysis.best_move_equity_error — NULL for a played move none of the
--     candidates names, where 0 claimed a perfect play.
--
-- All three are projections of the compressed analysis blob and of the
-- position's legal plays, which SQL cannot compute: the rows are rewritten by
-- the repair pass (RepairDenormalisedColumns, `blunderdb repair`). The
-- match_stats rows summarise those columns and are dropped here, so nothing
-- is served from the old counts once the repair has run.
-- Schema-visible: bumps domain.DatabaseVersion to 2.37.0.

DELETE FROM match_stats;
