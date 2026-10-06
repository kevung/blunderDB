package database

import "context"

// migrate_2_35_0_to_2_36_0 gives duel its is_open column: whether the Duel is
// being played, written in the row so every process on the library sees the
// same Duels open. Every draft already stored starts in suspense, as a
// restart left it when the open one lived in memory. A library from before
// the duel table reaches this step without it, which EnsureSchema creates
// afterwards with the column.
func (d *Database) migrate_2_35_0_to_2_36_0(context.Context) error {
	present, err := d.columnExists("duel", "id")
	if err != nil || !present {
		return err
	}
	return d.addColumn("duel", "is_open INTEGER NOT NULL DEFAULT 0")
}
