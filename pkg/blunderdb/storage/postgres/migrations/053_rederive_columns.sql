-- Forward migration: the 2.50.0 wave — every derived column computed again.
--
--   * The analysis columns and the per-move decision columns are derived
--     again, and match_stats emptied (cells and positions cascade): the
--     played move is matched to a candidate by the board it leaves, and
--     countedExpr changed. That pass needs Go: it is recountDecisions, which
--     runGoBackfills runs once more under goBackfillsGeneration "5".
--   * move.error_mp stays, inert: nothing writes or reads it any more, but a
--     daemon older than 2.50.0 still does. It goes once a minimum reading
--     version turns such binaries away.
-- Idempotent: a comment only.
-- Schema-visible: bumps domain.DatabaseVersion to 2.50.0.

COMMENT ON COLUMN move.error_mp IS 'Inert since 2.50.0: kept for older binaries; read move.decision_error_mp.';
