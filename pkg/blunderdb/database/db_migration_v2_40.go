package database

import "context"

// migrate_2_39_0_to_2_40_0 gives a Match its video (ADR-0082):
// match.video_source, and on each move the two Repères, roll_tick_ms and
// tick_ms. All are added NULL — no video, Repères unknown, never zero — for
// every Match already stored. A table this library still lacks is created
// with the columns by EnsureSchema afterwards.
func (d *Database) migrate_2_39_0_to_2_40_0(context.Context) error {
	for _, c := range []struct{ table, col string }{
		{"match", "video_source TEXT"},
		{"move", "roll_tick_ms INTEGER"},
		{"move", "tick_ms INTEGER"},
	} {
		present, err := d.columnExists(c.table, "id")
		if err != nil {
			return err
		}
		if !present {
			continue
		}
		if err := d.addColumn(c.table, c.col); err != nil {
			return err
		}
	}
	return nil
}
