package database

import (
	"context"
	"fmt"
	"strings"
)

// migrate_2_45_0_to_2_46_0 makes comment identifiers only grow. Without
// AUTOINCREMENT, SQLite gives a new row max(id)+1, so deleting the newest
// comment hands its identifier to the next one; an incremental export, which
// carries the comments above its mark (ADR-0096), would then never send that
// comment. The table is rebuilt with AUTOINCREMENT, its sequence starting at
// the highest identifier it holds; ids, dates, origins and authors are kept.
func (d *Database) migrate_2_45_0_to_2_46_0(ctx context.Context) error {
	var ddl string
	if err := d.db.QueryRowContext(ctx,
		`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'comment'`).Scan(&ddl); err != nil {
		return fmt.Errorf("comment: reading the table: %w", err)
	}
	if strings.Contains(strings.ToUpper(ddl), "AUTOINCREMENT") {
		return nil
	}
	return rebuildTable(ctx, d.db, "comment")
}
