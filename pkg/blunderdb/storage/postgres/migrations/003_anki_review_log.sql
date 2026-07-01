-- Forward migration: add the anki_review_log table that records every spaced
-- repetition review (rating + FSRS outcome). Append-only; powers retention and
-- streak statistics, the review heatmap and a faithful undo of the last review.
-- Idempotent, so it is safe on a fresh database (whose v2.7.0 baseline already
-- includes the table) and on existing databases bootstrapped before it existed.

CREATE TABLE IF NOT EXISTS anki_review_log (
    id              BIGSERIAL PRIMARY KEY,
    tenant_id       BIGINT NOT NULL,
    card_id         BIGINT NOT NULL REFERENCES anki_card(id) ON DELETE CASCADE,
    deck_id         BIGINT NOT NULL,
    position_id     BIGINT NOT NULL,
    rating          BIGINT NOT NULL,
    state           BIGINT NOT NULL DEFAULT 0,
    stability       DOUBLE PRECISION DEFAULT 0,
    difficulty      DOUBLE PRECISION DEFAULT 0,
    elapsed_days    BIGINT DEFAULT 0,
    scheduled_days  BIGINT DEFAULT 0,
    reviewed_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_anki_review_log_card ON anki_review_log (tenant_id, card_id, reviewed_at);
CREATE INDEX IF NOT EXISTS idx_anki_review_log_deck ON anki_review_log (tenant_id, deck_id, reviewed_at);

UPDATE metadata SET value = '2.11.0' WHERE key = 'database_version';
