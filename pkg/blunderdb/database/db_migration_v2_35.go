package database

import "context"

// migrate_2_34_0_to_2_35_0 gives match_origin what external Sides brought to
// a Duel: declared_bots, the Bots they declared playing behind them, ” for
// every origin already stored, where none was declared. A library from before
// match_origin reaches this step without the table, which EnsureSchema
// creates afterwards with the column.
func (d *Database) migrate_2_34_0_to_2_35_0(context.Context) error {
	present, err := d.columnExists("match_origin", "match_id")
	if err != nil || !present {
		return err
	}
	return d.addColumn("match_origin", "declared_bots TEXT NOT NULL DEFAULT ''")
}
