package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// migrate_2_50_0_to_2_51_0 gives each object a uuid column and its unique
// index: a stable UUID version 7, beside the integer id that stays the key.
// The identifiers of the rows already held, and the triggers that name the
// rows written from now on, come from sqlite.EnsureObjectUUIDs after the
// chain, once the tables created late exist.
func (d *Database) migrate_2_50_0_to_2_51_0(context.Context) error {
	for _, f := range sqlshared.ObjectUUIDFamilies {
		var n int
		if err := d.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, f.Table).Scan(&n); err != nil {
			return err
		}
		if n == 0 {
			continue
		}
		if err := d.addColumn(f.Table, "uuid TEXT"); err != nil {
			return err
		}
		if _, err := d.db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_` + f.Table + `_uuid ON ` + f.Table + `(uuid)`); err != nil {
			return fmt.Errorf("indexing %s.uuid: %w", f.Table, err)
		}
	}
	return nil
}
