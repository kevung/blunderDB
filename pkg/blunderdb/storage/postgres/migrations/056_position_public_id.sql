-- Forward migration: position.public_id, an opaque label the engine stores
-- and returns. A nullable column without a default is a catalogue-only change
-- in PostgreSQL: no row is rewritten, so no change_log entry is due and the
-- uuid-bearing rows keep their journal stamps.
-- No COMMIT, no role switch: it holds on blunderdb_user under FORCE RLS.
-- ALTER TABLE takes ACCESS EXCLUSIVE briefly: lock_timeout makes a busy table
-- fail the migration, retried at the next start, rather than queue every
-- reader behind a long transaction. Idempotent.
-- Schema-visible: bumps domain.DatabaseVersion to 2.53.0.

SET LOCAL lock_timeout = '15s';

ALTER TABLE "position" ADD COLUMN IF NOT EXISTS public_id TEXT;
