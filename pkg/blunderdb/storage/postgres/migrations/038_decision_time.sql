-- Forward migration: the 2.33.0 wave — the time of a Duel (ADR-0073).
--
--   * move.decision_ms, move.cube_decision_ms — how long the player took over
--     the Move's decision, and over the cube decision before a checker play's
--     roll, in milliseconds. Added NULL and left NULL: NULL is unknown, never
--     zero, and every Match already stored was not timed.
--   * match_origin.lost_on_time becomes over_time: the player (1 or 2) whose
--     reserve ran out first, 0 for none. A boolean could not say who, nor
--     note an overrun the Duel was set to play through. No row held a 1: no
--     Cadence existed before this wave.
-- Schema-visible: bumps domain.DatabaseVersion to 2.33.0.

ALTER TABLE move ADD COLUMN IF NOT EXISTS decision_ms      BIGINT;
ALTER TABLE move ADD COLUMN IF NOT EXISTS cube_decision_ms BIGINT;

DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema() AND table_name = 'match_origin'
          AND column_name = 'lost_on_time'
    ) THEN
        ALTER TABLE match_origin RENAME COLUMN lost_on_time TO over_time;
    END IF;
END $$;
