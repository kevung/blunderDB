package database

import (
	"context"
	"fmt"
)

// migrate_2_37_0_to_2_38_0 drops the materialised match statistics: their MWC
// converted a take or a pass at the doubled cube, where its equity is counted
// in units of the cube before the double. No stored column changes; the rows
// summarising them were computed by the old conversion and the next read
// recomputes them.
func (d *Database) migrate_2_37_0_to_2_38_0(ctx context.Context) error {
	present, err := d.columnExists("match_stats", "match_id")
	if err != nil || !present {
		return err
	}
	if _, err := d.db.ExecContext(ctx, `DELETE FROM match_stats`); err != nil {
		return fmt.Errorf("cube response MWC: invalidate match stats: %w", err)
	}
	return nil
}
