-- Forward migration: the "Sauf" (exclusion) structure of a search — the
-- checkers a matching position must NOT have — stored beside the include
-- position on both search_history and filter_library. SQLite gained the two
-- columns in its 2.7.0 → 2.8.0 migration (db_migration_v2_5.go); this backend
-- was bootstrapped from the 2.7.0 schema and never received them, so the
-- Database wrapper could not delegate SaveExcludePosition / SaveSearchHistory
-- to the storage contract.
--
-- No version bump: this is a catch-up of a change domain.DatabaseVersion has
-- covered since 2.8.0, not a new schema version. Recording one here would
-- desynchronise database_version from the Go constant (see migrations/README.md).
--
-- Idempotent, so it is safe on a fresh database whose 001 baseline already has
-- the columns.

ALTER TABLE filter_library ADD COLUMN IF NOT EXISTS exclude_position TEXT;
ALTER TABLE search_history ADD COLUMN IF NOT EXISTS exclude_position TEXT;
