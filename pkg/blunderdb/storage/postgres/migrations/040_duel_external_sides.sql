-- Forward migration: the 2.35.0 wave — what external Sides bring to a Duel
-- (ADR-0072 rules 8 and 10).
--
--   * match_origin.declared_bots — the Bots external Sides declared playing
--     behind them, as JSON, '' when none did. Declared by the client, never
--     attested by the Arbiter.
--   * match_origin.contributions — the Sides' contributions to a combined
--     seed, as JSON in player order, '' when the Duel rolled from its sealed
--     seed alone. Origins already stored read '' in both.
-- Schema-visible: bumps domain.DatabaseVersion to 2.35.0.

ALTER TABLE match_origin ADD COLUMN IF NOT EXISTS declared_bots TEXT NOT NULL DEFAULT '';
ALTER TABLE match_origin ADD COLUMN IF NOT EXISTS contributions TEXT NOT NULL DEFAULT '';
