-- Forward migration: the 2.50.0 wave — move.error_mp dropped, every derived
-- column computed again.
--
--   * move.error_mp — written only by an explicit pass, read only by the
--     player contrast, which reads move.decision_error_mp instead (kept with
--     every write of an analysis or a move). Dropped.
--   * The analysis columns and the per-move decision columns are derived
--     again, and match_stats emptied (cells and positions cascade): the
--     played move is matched to a candidate by the board it leaves, and
--     countedExpr changed. That pass needs Go: it is recountDecisions, which
--     runGoBackfills runs once more under goBackfillsGeneration "5".
-- Idempotent: DROP COLUMN IF EXISTS.
-- Schema-visible: bumps domain.DatabaseVersion to 2.50.0.

ALTER TABLE move DROP COLUMN IF EXISTS error_mp;
