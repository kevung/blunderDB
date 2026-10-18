-- Forward migration: the 2.43.0 wave — the date of a visit.
--
--   * match.last_visited_at — set with match.last_visited_position, so that
--     the last visited match is the one visited last, not the one imported
--     last. NULL on a match visited before this wave; it sorts after the
--     dated visits.
-- Schema-visible: bumps domain.DatabaseVersion to 2.43.0.

ALTER TABLE match ADD COLUMN IF NOT EXISTS last_visited_at TIMESTAMPTZ;
