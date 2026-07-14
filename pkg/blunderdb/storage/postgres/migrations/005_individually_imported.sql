-- Forward migration: add position.individually_imported (docs/adr/0001) and
-- backfill it from the only signal an existing database carries — a position
-- reachable from no move never came from a match.
--
-- The backfill reconstructs history once; from here on the flag is written at
-- import time and is exact. Its two error classes are accepted deliberately:
-- positions created by a cross-format "enrich" import are match-sourced yet
-- have no move row (false positives), and a position individually imported
-- before a match that also contained it is indistinguishable from a plain match
-- position (false negatives, unrecoverable — the information is not stored).
--
-- Idempotent, so it is safe on a fresh database whose 001 baseline already has
-- the column: the backfill then runs over an empty position table.

ALTER TABLE position ADD COLUMN IF NOT EXISTS individually_imported BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_position_individual ON position (tenant_id) WHERE individually_imported;

UPDATE position p
SET individually_imported = TRUE
WHERE NOT EXISTS (SELECT 1 FROM move m WHERE m.position_id = p.id);

UPDATE metadata SET value = '2.13.0' WHERE key = 'database_version';
