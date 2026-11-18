-- Forward migration: a search sorted by pips reads its positions in pip order
-- (no schema change a reader sees, DatabaseVersion untouched).
--
-- The search writes the sort as "pip_1 ASC|DESC, id" over the rows whose
-- pip_1 is set, and a structure as four mask conjuncts. Without an index in
-- that order, a structure sorted by pips ranks every matching row of the
-- tenant to return one page. Each index below holds the order, and the masks
-- in INCLUDE, so the page is read in order and filtered from the index alone.
-- The descending order keeps id ascending, which a backward scan of the first
-- index cannot give: two indexes. idx_position_pip_1 is a prefix of the first
-- and is dropped; a pip range reads the new index as it read the old one.
--
-- Back: DROP INDEX idx_position_pip_order, idx_position_pip_order_desc;
--   CREATE INDEX idx_position_pip_1 ON position (tenant_id, pip_1);
--
-- Each CREATE INDEX holds a SHARE lock on position (reads pass, writes wait)
-- until this file's transaction ends: two passes over the table. lock_timeout
-- makes a busy table fail the migration, retried at the next start, rather
-- than queue writers behind it. Idempotent. No COMMIT, no role switch, no
-- foreign key: it holds on blunderdb_user under FORCE RLS.

SET LOCAL lock_timeout = '15s';

CREATE INDEX IF NOT EXISTS idx_position_pip_order ON position (tenant_id, pip_1, id)
    INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2);
CREATE INDEX IF NOT EXISTS idx_position_pip_order_desc ON position (tenant_id, pip_1 DESC, id)
    INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2);
DROP INDEX IF EXISTS idx_position_pip_1;
