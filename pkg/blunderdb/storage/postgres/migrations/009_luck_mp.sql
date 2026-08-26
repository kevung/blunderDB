-- Forward migration: add move.luck_mp (docs/adr/0010) — the luck of a roll, in
-- signed millipoints of equity (positive = lucky), as the analysing tool
-- computed it. blunderDB has no evaluation engine and never computes it: it
-- carries the value found in the source file (eXtreme Gammon's ErrLuck, gnuBG's
-- LU property).
--
-- The column is NULLable with NO default and there is deliberately NO backfill.
-- Zero is a real value — a neutral roll — so an existing row must read
-- "unknown" rather than "neutral". Nothing on disk could feed a backfill
-- either: unlike the denormalised analysis columns, luck was never stored in
-- the analysis JSON. Existing rolls gain a value only when their match is
-- imported again from the source file.
--
-- It lands on move rather than position because a Position is deduplicated
-- across matches, while the luck of a roll is a fact of one occurrence of it.
--
-- Idempotent, so it is safe on a fresh database whose 001 baseline already has
-- the column.

ALTER TABLE move ADD COLUMN IF NOT EXISTS luck_mp BIGINT;

UPDATE metadata SET value = '2.15.0' WHERE key = 'database_version';
