package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// checkIssued refuses an id the AUTOINCREMENT counter of table has not handed
// out: an explicit id above it would raise the counter and skip the ids in
// between, so a Reinstate accepts only an id the store itself issued. table is
// one of this package's constant table names, never input.
func checkIssued(ctx context.Context, db execer, table string, id int64) (bool, error) {
	var issued int64
	err := db.QueryRowContext(ctx, `SELECT seq FROM sqlite_sequence WHERE name = ?`, table).Scan(&issued)
	if errors.Is(err, sql.ErrNoRows) {
		// No row yet: nothing was ever inserted under AUTOINCREMENT.
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read %s id counter: %w", table, err)
	}
	return id <= issued, nil
}
