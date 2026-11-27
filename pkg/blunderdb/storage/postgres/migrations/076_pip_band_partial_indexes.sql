-- Forward migration: the bands of a structure sorted by ascending pips are
-- read by partial indexes, one per band (no schema change a reader sees,
-- DatabaseVersion untouched).
--
-- idx_position_pip_band (074) keyed the band as width_bucket(occupancy_1, …).
-- The engine reads under RLS, where a condition calling a function that is not
-- leakproof (width_bucket) cannot be an index condition. A partial index whose
-- predicate compares occupancy_1 with constants is matched by the same
-- comparisons in the query, which are leakproof. Band n holds the rows whose
-- black farthest checker stands on point n - 1; the window starts past black's
-- farthest checker, so band 1 is never read and has no index.
--
-- Back: the DROP INDEX of the 25 indexes below, then 074_pip_band_index.sql.
--
-- CREATE INDEX holds a SHARE lock on position (reads pass, writes wait) until
-- this file's transaction ends. lock_timeout makes a busy table fail the
-- migration, retried at the next start. Idempotent. No COMMIT, no role
-- switch: it holds on blunderdb_user under FORCE RLS.

SET LOCAL lock_timeout = '15s';

CREATE INDEX IF NOT EXISTS idx_position_pip_band_02 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 2 AND occupancy_1 < 4;
CREATE INDEX IF NOT EXISTS idx_position_pip_band_03 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 4 AND occupancy_1 < 8;
CREATE INDEX IF NOT EXISTS idx_position_pip_band_04 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 8 AND occupancy_1 < 16;
CREATE INDEX IF NOT EXISTS idx_position_pip_band_05 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 16 AND occupancy_1 < 32;
CREATE INDEX IF NOT EXISTS idx_position_pip_band_06 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 32 AND occupancy_1 < 64;
CREATE INDEX IF NOT EXISTS idx_position_pip_band_07 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 64 AND occupancy_1 < 128;
CREATE INDEX IF NOT EXISTS idx_position_pip_band_08 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 128 AND occupancy_1 < 256;
CREATE INDEX IF NOT EXISTS idx_position_pip_band_09 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 256 AND occupancy_1 < 512;
CREATE INDEX IF NOT EXISTS idx_position_pip_band_10 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 512 AND occupancy_1 < 1024;
CREATE INDEX IF NOT EXISTS idx_position_pip_band_11 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 1024 AND occupancy_1 < 2048;
CREATE INDEX IF NOT EXISTS idx_position_pip_band_12 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 2048 AND occupancy_1 < 4096;
CREATE INDEX IF NOT EXISTS idx_position_pip_band_13 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 4096 AND occupancy_1 < 8192;
CREATE INDEX IF NOT EXISTS idx_position_pip_band_14 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 8192 AND occupancy_1 < 16384;
CREATE INDEX IF NOT EXISTS idx_position_pip_band_15 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 16384 AND occupancy_1 < 32768;
CREATE INDEX IF NOT EXISTS idx_position_pip_band_16 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 32768 AND occupancy_1 < 65536;
CREATE INDEX IF NOT EXISTS idx_position_pip_band_17 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 65536 AND occupancy_1 < 131072;
CREATE INDEX IF NOT EXISTS idx_position_pip_band_18 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 131072 AND occupancy_1 < 262144;
CREATE INDEX IF NOT EXISTS idx_position_pip_band_19 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 262144 AND occupancy_1 < 524288;
CREATE INDEX IF NOT EXISTS idx_position_pip_band_20 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 524288 AND occupancy_1 < 1048576;
CREATE INDEX IF NOT EXISTS idx_position_pip_band_21 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 1048576 AND occupancy_1 < 2097152;
CREATE INDEX IF NOT EXISTS idx_position_pip_band_22 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 2097152 AND occupancy_1 < 4194304;
CREATE INDEX IF NOT EXISTS idx_position_pip_band_23 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 4194304 AND occupancy_1 < 8388608;
CREATE INDEX IF NOT EXISTS idx_position_pip_band_24 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 8388608 AND occupancy_1 < 16777216;
CREATE INDEX IF NOT EXISTS idx_position_pip_band_25 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 16777216 AND occupancy_1 < 33554432;
CREATE INDEX IF NOT EXISTS idx_position_pip_band_26 ON position (tenant_id, pip_1, id) INCLUDE (occupancy_1, point_mask_1, occupancy_2, point_mask_2) WHERE occupancy_1 >= 33554432;

DROP INDEX IF EXISTS idx_position_pip_band;
