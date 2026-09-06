-- Forward migration: position.game_phase, the derived phase label (2.19.0,
-- issue #264, fiche I.8).
--
-- The phase — opening / middlegame / race / bearoff — is computed from the
-- board by engine.ClassifyGamePhase and written by every path that writes a
-- position. It is stored rather than computed at query time so a search can
-- narrow on it with an index instead of decoding every board.
--
-- Rows written before this migration keep 0 (unknown) until they are rewritten
-- or `blunderdb repair` recomputes them: the backfill cannot run in SQL, since
-- the classification reads the board out of the compact `state` encoding.
-- A search for a phase therefore returns nothing on an un-repaired database
-- rather than something wrong, and `blunderdb verify` counts the unknown rows.

ALTER TABLE position ADD COLUMN IF NOT EXISTS game_phase INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_position_game_phase ON position (tenant_id, game_phase);
