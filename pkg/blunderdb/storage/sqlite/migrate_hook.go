package sqlite

import (
	"context"
	"database/sql"
)

// Migrator upgrades a pre-existing (non-fresh) SQLite database in place to the
// current schema version. The legacy 1.0→current migration chain lives in
// package database (where its engine-backed helpers reside); that package
// registers it here from an init() so the storage backend can upgrade an old
// user database opened headless (serve / migrate / call) without importing
// database (which would be an import cycle).
//
// progress, when non-nil (Options.MigrationProgress), is invoked as the chain
// runs (see database.Database.SetMigrationProgress) with the backfill phases
// ("position", "analysis", …) that `blunderdb migrate` streams as
// schema-migration events and the desktop shows in its status bar while it
// opens a library.
type Migrator func(ctx context.Context, db *sql.DB, progress func(phase string, done, total int)) error

// registeredMigrator is set by package database's init via RegisterMigrator. It
// is nil for pure-library consumers that import only this package; Migrate then
// refuses a non-fresh database whose version is not current, and brings a
// current one to the schema's layout (EnsureSchema), which may recluster the
// derived match_stats tables.
var registeredMigrator Migrator

// RegisterMigrator installs the legacy migration chain. It is called once, from
// package database's init(); the last registration wins.
func RegisterMigrator(m Migrator) { registeredMigrator = m }
