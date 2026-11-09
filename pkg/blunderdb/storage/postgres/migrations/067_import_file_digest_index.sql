-- Forward migration: an import recognises a file it already imported by its
-- bytes, before decoding it (no schema change a reader sees, DatabaseVersion
-- untouched).
--
-- import_batch_file already journals each file's sha256 and the match it gave;
-- ImportBatchStore.KnownFile reads it by (tenant, digest) for the files of the
-- tenant's earlier batches. The partial index holds only the lines that name a
-- match: a deleted match unlinks its lines (ON DELETE SET NULL), so they drop
-- out of the index and the file imports again.
--
-- Back: DROP INDEX idx_import_batch_file_sha.
--
-- CREATE INDEX holds a SHARE lock on import_batch_file (one small row per file
-- ever imported) until this file's transaction ends. Idempotent. No COMMIT, no
-- role switch, no foreign key: it holds on blunderdb_user under FORCE RLS.

SET LOCAL lock_timeout = '15s';

CREATE INDEX IF NOT EXISTS idx_import_batch_file_sha
    ON import_batch_file (tenant_id, sha256)
    WHERE match_id IS NOT NULL AND sha256 <> '';
