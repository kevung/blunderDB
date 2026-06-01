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
type Migrator func(ctx context.Context, db *sql.DB) error

// registeredMigrator is set by package database's init via RegisterMigrator. It
// is nil for pure-library consumers that import only this package; in that case
// Migrate leaves a non-fresh database untouched (assumed already current).
var registeredMigrator Migrator

// RegisterMigrator installs the legacy migration chain. It is called once, from
// package database's init(); the last registration wins.
func RegisterMigrator(m Migrator) { registeredMigrator = m }
