-- Forward migration: index comment(tenant_id, position_id) for the
-- comment-presence search filter (`co` / `xco`, issue #109).
--
-- The filter is an EXISTS/NOT EXISTS subquery on the comment table, run once
-- per search alongside the position scan. comment.position_id carried no index
-- at all, so that subquery was a full comment scan. The index leads with
-- tenant_id because every query in this backend is tenant-scoped and the
-- subquery filters on both columns.
--
-- Index-only: no column, no table, no data changes. Nothing here is
-- schema-visible, so this migration deliberately does NOT bump database_version
-- in metadata and domain.DatabaseVersion is left alone — recording a version
-- with no counterpart in the Go constant would desynchronise the two.
--
-- Idempotent, so it is safe on a fresh database whose 001 baseline already
-- creates the index.

CREATE INDEX IF NOT EXISTS idx_comment_position ON comment (tenant_id, position_id);
