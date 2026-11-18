-- Forward migration: the collections of a position are read by the position
-- (no schema change a reader sees, DatabaseVersion untouched).
--
-- A table of positions lists, per row, the collections that hold it
-- (ListFieldsStore): one query for a page of ids, which without this index
-- reads every collection_position row of the tenant. The same index serves
-- the cascade from a deleted position through the (tenant_id, position_id)
-- foreign key, which also had none.
--
-- Back: DROP INDEX idx_collection_position_position.
--
-- CREATE INDEX holds a SHARE lock on collection_position (reads pass, writes
-- wait) until this file's transaction ends. lock_timeout makes a busy table
-- fail the migration, retried at the next start, rather than queue writers
-- behind it. Idempotent. No COMMIT, no role switch, no foreign key: it holds
-- on blunderdb_user under FORCE RLS.

SET LOCAL lock_timeout = '15s';

CREATE INDEX IF NOT EXISTS idx_collection_position_position ON collection_position (tenant_id, position_id);
