-- Forward migration: the 2.34.0 wave — the gammonNet a Bot played with
-- (ADR-0072 rule 10).
--
--   * match_origin.bot_engine — the gammonNet tag whose policy the Bot played
--     ("gammonNet v1.5.0"), '' when no Bot played. Origins already stored
--     read '': no record names the version they were played with.
-- Schema-visible: bumps domain.DatabaseVersion to 2.34.0.

ALTER TABLE match_origin ADD COLUMN IF NOT EXISTS bot_engine TEXT NOT NULL DEFAULT '';
