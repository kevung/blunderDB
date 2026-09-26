package database

import (
	"context"
	"database/sql"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// init registers the migration chain with storage/sqlite (which cannot import
// this package), so `serve`/`migrate`/`call` upgrade old databases too. It runs
// on a transient Database, cancelled by the caller's ctx; progress may be nil.
func init() {
	sqlite.RegisterMigrator(func(ctx context.Context, db *sql.DB, progress func(phase string, done, total int)) error {
		d := &Database{db: db}
		d.SetMigrationProgress(progress)
		return d.runMigrationChain(ctx)
	})
}
