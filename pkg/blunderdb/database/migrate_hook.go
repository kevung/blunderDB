package database

import (
	"context"
	"database/sql"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// init registers the legacy SQLite migration chain with the storage backend so
// that `serve` / `migrate` / `call` (which open databases through
// storage/sqlite, not the Database wrapper) can upgrade a pre-existing user
// database. The chain lives here because its helpers are engine-backed and
// already wired into this package; the storage package cannot import database
// (cycle), so it exposes a registration hook instead.
//
// The migrator runs on a transient Database bound to the storage handle:
// cancellation is driven by the caller-supplied ctx (the storage layer's
// context) rather than the GUI's CancelImport — the right behaviour for the
// headless path. progress forwards storage.Options.MigrationProgress (set by
// whoever opened the Storage) to the same SetMigrationProgress the GUI uses;
// nil is fine (emitMigrationProgress no-ops).
func init() {
	sqlite.RegisterMigrator(func(ctx context.Context, db *sql.DB, progress func(phase string, done, total int)) error {
		d := &Database{db: db}
		d.SetMigrationProgress(progress)
		return d.runMigrationChain(ctx)
	})
}
