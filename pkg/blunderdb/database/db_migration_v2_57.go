package database

import (
	"context"
	"fmt"
)

// migrate_2_56_0_to_2_57_0 gives the Anki deck and the tournament a uuid
// column and its unique index, as migrate_2_50_0_to_2_51_0 did for the other
// objects. The identifiers of the rows already held, and the journal triggers
// of the two tables, come from sqlite.EnsureObjectUUIDs after the chain.
func (d *Database) migrate_2_56_0_to_2_57_0(context.Context) error {
	for _, table := range []string{"anki_deck", "tournament"} {
		var n int
		if err := d.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&n); err != nil {
			return err
		}
		if n == 0 {
			continue
		}
		if err := d.addColumn(table, "uuid TEXT"); err != nil {
			return err
		}
		if _, err := d.db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_` + table + `_uuid ON ` + table + `(uuid)`); err != nil {
			return fmt.Errorf("indexing %s.uuid: %w", table, err)
		}
	}
	return nil
}
