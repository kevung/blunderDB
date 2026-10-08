-- Forward migration: the 2.40.0 wave — the video of a Match (ADR-0082).
--
--   * match.video_source — the video the Match was transcribed from: an
--     http(s) URL or a local path. NULL when none is attached.
--   * move.roll_tick_ms, move.tick_ms — the Move's Repères in that video, in
--     milliseconds from the start of the media: the instant the dice fell and
--     the instant the action was done. NULL is unknown, never zero.
-- Schema-visible: bumps domain.DatabaseVersion to 2.40.0.

ALTER TABLE match ADD COLUMN IF NOT EXISTS video_source TEXT;
ALTER TABLE move  ADD COLUMN IF NOT EXISTS roll_tick_ms BIGINT;
ALTER TABLE move  ADD COLUMN IF NOT EXISTS tick_ms      BIGINT;
