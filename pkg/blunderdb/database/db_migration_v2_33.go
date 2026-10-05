package database

import (
	"context"
	"fmt"
)

// migrate_2_32_0_to_2_33_0 gives the move table the time a Duel measures
// (ADR-0073): decision_ms and cube_decision_ms, added NULL — unknown, never
// zero, for every Match already stored. match_origin.lost_on_time becomes
// over_time, the player whose reserve ran out: no row held anything but 0,
// no Cadence having existed. A 2.31.0 library reaches this step without
// match_origin, which EnsureSchema creates afterwards with the new column.
func (d *Database) migrate_2_32_0_to_2_33_0(context.Context) error {
	for _, col := range []string{"decision_ms INTEGER", "cube_decision_ms INTEGER"} {
		if err := d.addColumn("move", col); err != nil {
			return err
		}
	}
	old, err := d.columnExists("match_origin", "lost_on_time")
	if err != nil || !old {
		return err
	}
	if _, err := d.db.Exec(`ALTER TABLE match_origin RENAME COLUMN lost_on_time TO over_time`); err != nil {
		return fmt.Errorf("renaming match_origin.lost_on_time: %w", err)
	}
	return nil
}
