package database

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// rederivePendingKey marks, in metadata, a library whose derived columns the
// 2.50.0 step has asked to be computed again: the pass needs the tables and
// columns ensureAllTablesExist creates after the chain, and the mark keeps it
// owed across an open interrupted before it ends. Never exported (not in
// issuance.CarriedMetadataKeys).
const rederivePendingKey = "rederive_columns_pending"

// migrate_2_49_0_to_2_50_0 owes the library a new derivation of every column
// the analyses and the moves project from their blobs, drops move.error_mp,
// and empties match_stats.
//
// The played move is now matched to a candidate by the board it leaves
// (engine.PlayedCandidate), no longer by its notation alone, so the stored
// best_move_equity_error and move.decision_error_mp may price another play;
// and analyses brought in by a library import before the import derived its
// columns have none. finishRederivedColumns rewrites both after the chain.
// match_stats summarises those columns under countedExpr, whose rule changed
// too: its rows go whatever the repair finds, the cells and positions with
// them (ON DELETE CASCADE), and the next read recomputes them.
//
// move.error_mp was only ever written by an explicit pass and read by the
// player contrast, which reads move.decision_error_mp instead: kept up to
// date with every write of an analysis or a move.
func (d *Database) migrate_2_49_0_to_2_50_0(ctx context.Context) error {
	if _, err := d.db.ExecContext(ctx, `INSERT OR REPLACE INTO metadata (key, value) VALUES (?, '1')`, rederivePendingKey); err != nil {
		return fmt.Errorf("marking the derived columns owed: %w", err)
	}

	present, err := d.columnExists("move", "error_mp")
	if err != nil {
		return err
	}
	if present {
		if _, err := d.db.ExecContext(ctx, `ALTER TABLE move DROP COLUMN error_mp`); err != nil {
			return fmt.Errorf("dropping move.error_mp: %w", err)
		}
	}

	present, err = d.columnExists("match_stats", "match_id")
	if err != nil || !present {
		return err
	}
	if _, err := d.db.ExecContext(ctx, `DELETE FROM match_stats`); err != nil {
		return fmt.Errorf("invalidate match stats: %w", err)
	}
	return nil
}

// finishRederivedColumns runs the derivation the 2.50.0 step owes, once the
// schema is whole, and clears the mark. One probe when nothing is owed.
func (d *Database) finishRederivedColumns(ctx context.Context) error {
	var owed int
	if err := d.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM metadata WHERE key = ?`, rederivePendingKey).Scan(&owed); err != nil {
		return fmt.Errorf("probing the derived columns owed: %w", err)
	}
	if owed == 0 {
		return nil
	}
	repaired, err := sqlite.New(d.db).Analyses().RepairDenormalisedColumns(ctx, "")
	if err != nil {
		return fmt.Errorf("deriving again the analysis columns: %w", err)
	}
	if _, err := d.db.ExecContext(ctx, `DELETE FROM metadata WHERE key = ?`, rederivePendingKey); err != nil {
		return fmt.Errorf("clearing the derived columns owed: %w", err)
	}
	slog.Info("derived again the columns of the stored analyses and moves", "rows", repaired)
	return nil
}
