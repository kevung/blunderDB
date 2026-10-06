-- Forward migration: the 2.35.0 wave — what external Sides bring to a Duel
-- (ADR-0072 rules 8 and 10).
--
--   * match_origin.declared_bots — the Bots external Sides declared playing
--     behind them, as JSON, '' when none did. Declared by the client, never
--     attested by the Arbiter. Origins already stored read ''.
-- Schema-visible: bumps domain.DatabaseVersion to 2.35.0.

ALTER TABLE match_origin ADD COLUMN IF NOT EXISTS declared_bots TEXT NOT NULL DEFAULT '';
