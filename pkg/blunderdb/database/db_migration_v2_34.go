package database

import "context"

// migrate_2_33_0_to_2_34_0 gives match_origin the gammonNet a Bot played
// with: bot_engine, the tag whose policy it played, empty for every origin
// already stored, which no record names. A library from before match_origin
// reaches this step without the table, which EnsureSchema creates afterwards
// with the column.
func (d *Database) migrate_2_33_0_to_2_34_0(context.Context) error {
	present, err := d.columnExists("match_origin", "match_id")
	if err != nil || !present {
		return err
	}
	return d.addColumn("match_origin", "bot_engine TEXT NOT NULL DEFAULT ''")
}
