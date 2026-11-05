-- Forward migration: one key where there were two (no schema change a reader
-- sees, DatabaseVersion untouched).
--
-- position and move carried a primary key on (id) next to a unique key on
-- (tenant_id, id), which every foreign key names. analysis carried a unique key
-- on (position_id) next to an index on (tenant_id, position_id). Every read runs
-- under row-level security or names its tenant, so the longer key reaches the
-- same row. analysis_pkey stays: maintenance walks every tenant's analyses by id.
-- idx_analysis_creation_date goes too: no statement reads it.
--
-- Back: ALTER TABLE position ADD PRIMARY KEY (id); the same for move;
-- CREATE UNIQUE INDEX idx_analysis_position ON analysis (position_id).
--
-- ALTER TABLE holds ACCESS EXCLUSIVE until this file's transaction ends, but the
-- building happens on none of the three: the index is already there, except for
-- the unique one on analysis, built once, in seconds to minutes with the table.
-- Idempotent. No COMMIT, no role switch: it holds on blunderdb_user under FORCE RLS.

SET LOCAL lock_timeout = '15s';

DROP INDEX IF EXISTS idx_analysis_creation_date;

CREATE UNIQUE INDEX IF NOT EXISTS idx_analysis_tenant_position_key ON analysis (tenant_id, position_id);
DROP INDEX IF EXISTS idx_analysis_tenant_position;
DROP INDEX IF EXISTS idx_analysis_position;

ALTER TABLE position DROP CONSTRAINT IF EXISTS position_pkey;
ALTER TABLE move DROP CONSTRAINT IF EXISTS move_pkey;

-- Keys that only grow: a page split on the right leaves the page full rather
-- than 90 % (1.4 GB on the largest library). Applies to the pages written from
-- now on; a REINDEX after a bulk load packs the rest.
ALTER INDEX position_tenant_id_key SET (fillfactor = 100);
ALTER INDEX move_tenant_id_key SET (fillfactor = 100);
ALTER INDEX analysis_pkey SET (fillfactor = 100);
ALTER INDEX idx_analysis_tenant_position_key SET (fillfactor = 100);
