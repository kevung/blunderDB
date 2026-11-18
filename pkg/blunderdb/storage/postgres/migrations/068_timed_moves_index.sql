-- Forward migration: the `tm` search filter reads the timed moves alone (no
-- schema change a reader sees, DatabaseVersion untouched).
--
-- The filter is an EXISTS on move per position, for a move of player 1 whose
-- decision_ms or cube_decision_ms is inside the bounds. Few libraries time
-- their moves: without this index a tenant that timed none still probed the
-- moves of each of its positions (2 s on 15.6 million). The partial index
-- holds the timed moves only (8 kB on that tenant); the planner proves the
-- filter's bounds imply its predicate and reads it for the statement as it is.
-- tenant_id leads: row-level security reads it, and one tenant never walks
-- another's entries.
--
-- Back: DROP INDEX idx_move_timed.
--
-- CREATE INDEX holds a SHARE lock on move (reads pass, writes wait) until this
-- file's transaction ends: one pass over the table, seconds on 17 million
-- moves. lock_timeout makes a busy table fail the migration, retried at the
-- next start, rather than queue writers behind it. Idempotent. No COMMIT, no
-- role switch, no foreign key: it holds on blunderdb_user under FORCE RLS.

SET LOCAL lock_timeout = '15s';

CREATE INDEX IF NOT EXISTS idx_move_timed ON move (tenant_id, position_id) WHERE decision_ms IS NOT NULL OR cube_decision_ms IS NOT NULL;
