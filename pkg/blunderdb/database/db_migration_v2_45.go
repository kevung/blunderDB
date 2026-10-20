package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// migrate_2_44_0_to_2_45_0 gives "no date of play" its one stored form, NULL.
// Rows written before that rule hold an empty string or the zero time as text (an edit
// that echoed the year-1 day back), which sorted as the oldest match instead
// of by import date. The positions those matches reach are re-dated, since
// position.match_date was derived from the year-1 day.
func (d *Database) migrate_2_44_0_to_2_45_0(ctx context.Context) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	undated := func(col string) string { return col + ` IS NOT NULL AND ` + domain.MatchUndatedSQLite(col) }
	ids, err := undatedMatchPositions(ctx, tx, undated("m.match_date"))
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE match SET match_date = NULL WHERE `+undated("match_date")); err != nil {
		return fmt.Errorf("null undated matches: %w", err)
	}
	if err := sqlite.RefreshPositionMatchDates(ctx, tx, ids); err != nil {
		return err
	}
	return tx.Commit()
}

// undatedMatchPositions lists the positions reached by the matches pred holds.
func undatedMatchPositions(ctx context.Context, tx *sql.Tx, pred string) ([]int64, error) {
	rows, err := tx.QueryContext(ctx, `SELECT DISTINCT mv.position_id
		FROM move mv JOIN game g ON g.id = mv.game_id JOIN match m ON m.id = g.match_id
		WHERE mv.position_id IS NOT NULL AND `+pred)
	if err != nil {
		return nil, fmt.Errorf("list positions of undated matches: %w", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
