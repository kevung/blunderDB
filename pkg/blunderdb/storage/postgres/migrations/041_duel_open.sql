-- Forward migration: the 2.36.0 wave — several Duels open at once (ADR-0072
-- rule 10).
--
--   * duel.is_open — 1 while the Duel is being played, its clocks running;
--     0 in suspense. In the row, so every daemon on the database sees the
--     same Duels open. Drafts already stored start in suspense.
-- Schema-visible: bumps domain.DatabaseVersion to 2.36.0.

ALTER TABLE duel ADD COLUMN IF NOT EXISTS is_open INTEGER NOT NULL DEFAULT 0;
