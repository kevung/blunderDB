-- Forward migration: the 2.38.0 wave — a take or a pass is converted to MWC
-- at the cube before the double, the unit its equity is counted in, no
-- longer at the doubled cube its position stores.
--
-- No stored column changes: the conversion runs when match_stats is filled.
-- The rows filled under the old conversion are dropped, so the next read
-- recomputes them.
-- Schema-visible: bumps domain.DatabaseVersion to 2.38.0.

DELETE FROM match_stats;
