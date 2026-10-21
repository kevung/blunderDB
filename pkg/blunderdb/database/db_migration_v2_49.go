package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// migrate_2_48_0_to_2_49_0 gives position its multi_played indicator: the
// column, filled from the moves already written, and the triggers on move
// that keep it from then on. A move-error search reads it instead of
// comparing the moves of every row it scans.
func (d *Database) migrate_2_48_0_to_2_49_0(ctx context.Context) error {
	present, err := d.columnExists("position", "multi_played")
	if err != nil {
		return err
	}
	if !present {
		if err := d.addColumn("position", "multi_played INTEGER NOT NULL DEFAULT 0"); err != nil {
			return err
		}
	}
	if _, err := d.db.ExecContext(ctx, sqlite.BackfillMultiPlayedSQL); err != nil {
		return fmt.Errorf("filling position.multi_played: %w", err)
	}
	return sqlite.EnsureMultiPlayedTriggers(ctx, d.db)
}
