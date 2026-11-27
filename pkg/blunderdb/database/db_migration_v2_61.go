package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// bestMovePendingKey marks, in metadata, that the 2.61 step's backfill of
// analysis.best_move has not run to its end yet.
const bestMovePendingKey = "best_move_backfill_pending"

// migrate_2_60_0_to_2_61_0 gives an analysis its best_move column
// (engine.BestMove), which a move-pattern search narrows by, and owes the
// pass that derives it (finishBestMoveBackfill). Until then NULL is "not
// derived yet": the search lets the row through and decides on the blob. No
// reader floor: an older binary that rewrites the blob has the column cleared
// by analysis_best_move_stale (sqlite.EnsureBestMoveTrigger).
func (d *Database) migrate_2_60_0_to_2_61_0(ctx context.Context) error {
	if err := d.addColumn("analysis", "best_move TEXT"); err != nil {
		return err
	}
	_, err := d.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, '1')`, bestMovePendingKey)
	return err
}

// bestMoveBackfiller is the SQLite analysis store's best-move pass.
type bestMoveBackfiller interface {
	BackfillBestMove(ctx context.Context) (int, error)
}

// finishBestMoveBackfill runs the pass the 2.61 step owes, once the schema is
// whole. One probe when nothing is owed; the pass resumes on the rows still
// NULL after an interrupted open.
func (d *Database) finishBestMoveBackfill(ctx context.Context) error {
	var mark string
	err := d.db.QueryRowContext(ctx, `SELECT value FROM metadata WHERE key = ?`, bestMovePendingKey).Scan(&mark)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("probing the best moves owed: %w", err)
	}
	if err := sqlite.EnsureBestMoveTrigger(ctx, d.db); err != nil {
		return err
	}
	b, ok := sqlite.New(d.db).Analyses().(bestMoveBackfiller)
	if !ok {
		return errors.New("deriving the best moves: the SQLite store has no backfill")
	}
	if _, err := b.BackfillBestMove(ctx); err != nil {
		return fmt.Errorf("deriving the best moves of the stored analyses: %w", err)
	}
	if _, err := d.db.ExecContext(ctx, `DELETE FROM metadata WHERE key = ?`, bestMovePendingKey); err != nil {
		return fmt.Errorf("clearing the best moves owed: %w", err)
	}
	return nil
}
