-- Forward migration: how many cards one review sitting serves, per deck
-- (ADR-0026 rule 2). SQLite gains the same column in its 2.15.0 → 2.16.0
-- migration (db_migration_v2_5.go).
--
-- Nullable with no default on purpose. NULL is "no limit" and is what every
-- deck that predates this column keeps, so the migration changes no behaviour;
-- 0 is a different state — it serves no card at all, which is a real use
-- (freezing a deck while preparing for a tournament) and the conflation this
-- column deliberately avoids.
--
-- A session, not a day: a deck here is a finite corpus, so nothing in this
-- schema needs a notion of "day", a rollover hour, or a timezone.

ALTER TABLE anki_deck ADD COLUMN IF NOT EXISTS session_limit INTEGER;
