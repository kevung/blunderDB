-- Forward migration: the blot filters of the search are answered in SQL (no
-- schema change a reader sees, DatabaseVersion untouched).
--
-- A blot is a point a side occupies (occupancy_N) and has not made
-- (point_mask_N); bit i of the masks is point i. The outfield is points 7 to
-- 18 for both sides (mask 524160); the jan is points 1 to 6 for player 1
-- (mask 126) and points 19 to 24 for player 2 (mask 33030144). The search
-- writes the count as below and its bound in clear, so a filter asking for at
-- least as many blots as an index's floor reads that index in id order: the
-- rare counts, which a walk of the tenant in id order would look for through
-- millions of rows (4 outfield blots or more: 0.6 % of 15.6 million positions,
-- 3 jan blots or more: 0.9 %). Below the floor the walk finds its page at once.
-- tenant_id leads: row-level security reads it, and one tenant never walks
-- another's entries. The text of each predicate is the one Dialect.BitsSet
-- writes (storage/postgres/adapter.go): keep the two alike.
--
-- Back: DROP INDEX idx_position_blot_o1, idx_position_blot_o2,
-- idx_position_blot_j1, idx_position_blot_j2.
--
-- Each CREATE INDEX holds a SHARE lock on position (reads pass, writes wait)
-- until this file's transaction ends: four passes over the table, 6 s on 15.6
-- million positions, 14 MB in all. lock_timeout makes a busy table fail the
-- migration, retried at the next start, rather than queue writers behind it.
-- Idempotent. No COMMIT, no role switch, no foreign key: it holds on
-- blunderdb_user under FORCE RLS.

SET LOCAL lock_timeout = '15s';

CREATE INDEX IF NOT EXISTS idx_position_blot_o1 ON position (tenant_id, id) WHERE bit_count(((occupancy_1 & ~point_mask_1) & 524160)::bit(32)) >= 4;
CREATE INDEX IF NOT EXISTS idx_position_blot_o2 ON position (tenant_id, id) WHERE bit_count(((occupancy_2 & ~point_mask_2) & 524160)::bit(32)) >= 4;
CREATE INDEX IF NOT EXISTS idx_position_blot_j1 ON position (tenant_id, id) WHERE bit_count(((occupancy_1 & ~point_mask_1) & 126)::bit(32)) >= 3;
CREATE INDEX IF NOT EXISTS idx_position_blot_j2 ON position (tenant_id, id) WHERE bit_count(((occupancy_2 & ~point_mask_2) & 33030144)::bit(32)) >= 3;
