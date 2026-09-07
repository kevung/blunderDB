-- Forward migration: two composite foreign keys learn WHICH column they null.
--
-- `ON DELETE SET NULL` on a composite foreign key nulls EVERY referencing
-- column, `tenant_id` included — and `tenant_id` is NOT NULL everywhere. So
-- the delete does not unlink the child, it fails:
--
--     null value in column "tenant_id" of relation "transcription"
--     violates not-null constraint
--
-- 001 and 017 knew this and wrote the column list PostgreSQL 15 gained
-- (`SET NULL (tournament_id)`, `SET NULL (position_id)`); 019 and 021 did
-- not, so deleting an import batch or a saved match hit the wall above
-- instead of doing what the two tables promise:
--
--   * `match.import_batch_id` (019, #257) — deleting a batch must not delete
--     the matches it brought in; they simply stop naming a batch.
--   * `transcription.match_id` (021, ADR-0045) — deleting the saved match
--     must not destroy the typing that produced it; the draft simply becomes
--     one that was never saved.
--
-- Constraint-only, like 006/008/010/017: no column, no data, nothing
-- schema-visible. `domain.DatabaseVersion` is left alone.
--
-- DROP then ADD rather than a guarded ADD: the constraint is there under the
-- right name with the wrong rule, so the guard the other migrations use
-- ("add it if it is missing") would find it and change nothing.

DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'match_import_batch_fkey') THEN
        ALTER TABLE match DROP CONSTRAINT match_import_batch_fkey;
    END IF;
END $$;

ALTER TABLE match ADD CONSTRAINT match_import_batch_fkey
    FOREIGN KEY (tenant_id, import_batch_id)
    REFERENCES import_batch (tenant_id, id) ON DELETE SET NULL (import_batch_id);

DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'transcription_match_fkey') THEN
        ALTER TABLE transcription DROP CONSTRAINT transcription_match_fkey;
    END IF;
END $$;

ALTER TABLE transcription ADD CONSTRAINT transcription_match_fkey
    FOREIGN KEY (tenant_id, match_id)
    REFERENCES match (tenant_id, id) ON DELETE SET NULL (match_id);
