package sqlite

import (
	"context"
	"strings"
	"sync"
)

// columnProbe tells whether a table has a column. A file read as it stands
// (OpenReadOnly, or the desktop before Migrate) can predate the newer columns;
// reading it must neither fail on them nor add them. A column found present is
// remembered, since it stays; an absent one is asked again, the file may be
// migrated meanwhile. A nil probe asks every time.
type columnProbe struct{ seen sync.Map }

func (p *columnProbe) has(ctx context.Context, db execer, table, column string) bool {
	key := table + "." + column
	if p != nil {
		if _, ok := p.seen.Load(key); ok {
			return true
		}
	}
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, column).Scan(&n); err != nil || n == 0 {
		return false
	}
	if p != nil {
		p.seen.Store(key, struct{}{})
	}
	return true
}

// matchColumns is matchSelectCols, the result read as unknown (no final
// score, no winner) from a file whose match table predates it.
func matchColumns(ctx context.Context, db execer, cols *columnProbe) string {
	if cols.has(ctx, db, "match", "final_score_1") {
		return matchSelectCols
	}
	return strings.Replace(matchSelectCols, "m.final_score_1, m.final_score_2, COALESCE(m.winner,0)", "NULL, NULL, 0", 1)
}
