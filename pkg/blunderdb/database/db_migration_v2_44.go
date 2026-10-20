package database

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// migrate_2_43_0_to_2_44_0 caches the blunder count: the triggers that delete
// the cached row on any write to the tables it is counted from. No row is
// written here; the first count after the migration fills it.
func (d *Database) migrate_2_43_0_to_2_44_0(ctx context.Context) error {
	return sqlite.EnsureBlunderCountTriggers(ctx, d.db)
}
