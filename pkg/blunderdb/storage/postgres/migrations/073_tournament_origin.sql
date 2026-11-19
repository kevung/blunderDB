-- Forward migration: a tournament states where it was played
-- (domain.Tournament.Origin), as free text the user or the producing
-- application chooses ("imported", "platform", "club"…).
--
--   * tournament.origin — '' when unknown, which every existing tournament
--     gets: nothing in its row tells an imported one from one created by hand.
--
-- The origin belongs to the tournament and travels with it: it is not a local
-- column, so the journal trigger of 060_deck_tournament_uuid.sql, which
-- compares whole rows, already dates a change of it.
-- Idempotent.
-- Schema-visible: bumps domain.DatabaseVersion to 2.60.0.

ALTER TABLE tournament ADD COLUMN IF NOT EXISTS origin TEXT NOT NULL DEFAULT '';
