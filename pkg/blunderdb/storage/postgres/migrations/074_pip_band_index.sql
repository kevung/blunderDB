-- Forward migration: a structure sorted by ascending pips reads, band by band,
-- only the positions whose farthest black checker stands where the structure
-- puts one (no schema change a reader sees, DatabaseVersion untouched).
--
-- In pip order the first rows are bear-offs: a structure that holds a point
-- outside the home board meets none of them, yet idx_position_pip_order makes
-- the search test them one by one before its first match. The band is the
-- farthest point black occupies, width_bucket of occupancy_1 over the powers
-- of two (0 for an empty side, n + 1 for a farthest checker on point n). A
-- structure with a black checker on point n keeps the bands above n only, and
-- each band is read in pip order from the least pips its checkers can add up
-- to (sqlshared: pipBandWindow). The expression is the adapter's PipBand:
-- the planner reads this index for that expression only.
--
-- Back: DROP INDEX idx_position_pip_band;
--
-- CREATE INDEX holds a SHARE lock on position (reads pass, writes wait) until
-- this file's transaction ends. lock_timeout makes a busy table fail the
-- migration, retried at the next start. Idempotent. No COMMIT, no role
-- switch: it holds on blunderdb_user under FORCE RLS.

SET LOCAL lock_timeout = '15s';

-- One line: the index parity test reads a declaration line by line.
CREATE INDEX IF NOT EXISTS idx_position_pip_band ON position (tenant_id, width_bucket(occupancy_1, '{1,2,4,8,16,32,64,128,256,512,1024,2048,4096,8192,16384,32768,65536,131072,262144,524288,1048576,2097152,4194304,8388608,16777216,33554432}'::integer[]), pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2);
