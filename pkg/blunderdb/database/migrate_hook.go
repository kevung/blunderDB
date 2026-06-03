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
// The migrator runs on a transient Database bound to the storage handle: it has
// no progress callback, and cancellation is driven by the caller-supplied ctx
// (the storage layer's context) rather than the GUI's CancelImport — the right
// behaviour for the headless path.
func init() {
	sqlite.RegisterMigrator(func(ctx context.Context, db *sql.DB) error {
		return (&Database{db: db}).runMigrationChain(ctx)
	})
}
