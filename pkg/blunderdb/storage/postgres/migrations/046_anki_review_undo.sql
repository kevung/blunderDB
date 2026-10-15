-- Forward migration: the 2.42.0 wave — undoing the last Anki review.
--
--   * anki_review_log.prev_* — the card's scheduling state just before the
--     review (due, stability, difficulty, elapsed and scheduled days, reps,
--     lapses, state, last review), copied from anki_card in the review's
--     transaction so an undo restores it exactly. NULL on every review logged
--     before this wave: its prior state was never kept and it cannot be
--     undone.
-- Schema-visible: bumps domain.DatabaseVersion to 2.42.0.

ALTER TABLE anki_review_log ADD COLUMN IF NOT EXISTS prev_due            TIMESTAMPTZ;
ALTER TABLE anki_review_log ADD COLUMN IF NOT EXISTS prev_stability      DOUBLE PRECISION;
ALTER TABLE anki_review_log ADD COLUMN IF NOT EXISTS prev_difficulty     DOUBLE PRECISION;
ALTER TABLE anki_review_log ADD COLUMN IF NOT EXISTS prev_elapsed_days   BIGINT;
ALTER TABLE anki_review_log ADD COLUMN IF NOT EXISTS prev_scheduled_days BIGINT;
ALTER TABLE anki_review_log ADD COLUMN IF NOT EXISTS prev_reps           BIGINT;
ALTER TABLE anki_review_log ADD COLUMN IF NOT EXISTS prev_lapses         BIGINT;
ALTER TABLE anki_review_log ADD COLUMN IF NOT EXISTS prev_state          BIGINT;
ALTER TABLE anki_review_log ADD COLUMN IF NOT EXISTS prev_last_review    TIMESTAMPTZ;
