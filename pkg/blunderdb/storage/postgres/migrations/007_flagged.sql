-- Forward migration: add position.flagged (docs/adr/0006) — the mark the user
-- put on a position in the tool the match came from. Today only eXtreme Gammon
-- produces it, recorded per move in the source file.
--
-- There is deliberately NO backfill. Unlike 005's individually_imported, which
-- could be reconstructed from the move graph, nothing in an existing database
-- records a source-file flag: it exists only in the .xg files. Existing
-- positions start unflagged and gain the mark when their match is imported
-- again — which is why ingest applies flags even to an exact duplicate whose
-- match rows are otherwise skipped.
--
-- Idempotent, so it is safe on a fresh database whose 001 baseline already has
-- the column.

ALTER TABLE position ADD COLUMN IF NOT EXISTS flagged BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_position_flagged ON position (tenant_id) WHERE flagged;

