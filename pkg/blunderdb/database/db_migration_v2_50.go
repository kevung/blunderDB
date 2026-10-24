package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// rederivePendingKey marks, in metadata, a library whose derived columns the
// 2.50.0 step has asked to be computed again. Its value is the id of the last
// analysis already derived (0 before the first page): the pass needs the
// tables and columns ensureAllTablesExist creates after the chain, and an
// open interrupted before it ends resumes past the pages it wrote. Never
// exported (not in issuance.CarriedMetadataKeys).
const rederivePendingKey = "rederive_columns_pending"

// rederivePhase names the pass in the migration progress events.
const rederivePhase = "rederive_columns"

// migrate_2_49_0_to_2_50_0 owes the library a new derivation of every column
// the analyses and the moves project from their blobs, and empties
// match_stats.
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
// move.error_mp is left in place, inert: an older blunderDB opens a newer
// library unchanged and still writes and reads it (schema_sqlite.go).
func (d *Database) migrate_2_49_0_to_2_50_0(ctx context.Context) error {
	if _, err := d.db.ExecContext(ctx, `INSERT OR IGNORE INTO metadata (key, value) VALUES (?, '0')`, rederivePendingKey); err != nil {
		return fmt.Errorf("marking the derived columns owed: %w", err)
	}
	present, err := d.columnExists("match_stats", "match_id")
	if err != nil || !present {
		return err
	}
	if _, err := d.db.ExecContext(ctx, `DELETE FROM match_stats`); err != nil {
		return fmt.Errorf("invalidate match stats: %w", err)
	}
	return nil
}

// pagedRepairer is the SQLite analysis store's resumable repair.
type pagedRepairer interface {
	RepairDenormalisedFrom(ctx context.Context, after int64, done func(lastID int64, repaired int) error) (int, error)
}

// finishRederivedColumns runs the derivation the 2.50.0 step owes, once the
// schema is whole, from the analysis the mark names; each page written moves
// the mark and reports progress (rederivePhase, in analysis ids). The mark
// goes once the last page is written. One probe when nothing is owed.
func (d *Database) finishRederivedColumns(ctx context.Context) error {
	var mark string
	err := d.db.QueryRowContext(ctx, `SELECT value FROM metadata WHERE key = ?`, rederivePendingKey).Scan(&mark)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("probing the derived columns owed: %w", err)
	}
	after, err := strconv.ParseInt(mark, 10, 64)
	if err != nil {
		after = 0 // an unreadable mark owes the whole pass
	}
	var maxID int64
	if err := d.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(id), 0) FROM analysis`).Scan(&maxID); err != nil {
		return fmt.Errorf("probing the analyses to derive: %w", err)
	}
	repairer, ok := sqlite.New(d.db).Analyses().(pagedRepairer)
	if !ok {
		return errors.New("deriving again the analysis columns: the SQLite store cannot resume a repair")
	}
	d.emitMigrationProgress(rederivePhase, int(after), int(maxID))
	repaired, err := repairer.RepairDenormalisedFrom(ctx, after, func(lastID int64, _ int) error {
		if _, err := d.db.ExecContext(ctx, `UPDATE metadata SET value = ? WHERE key = ?`, strconv.FormatInt(lastID, 10), rederivePendingKey); err != nil {
			return fmt.Errorf("recording the derived columns done: %w", err)
		}
		d.emitMigrationProgress(rederivePhase, int(lastID), int(maxID))
		return nil
	})
	if err != nil {
		return fmt.Errorf("deriving again the analysis columns: %w", err)
	}
	if _, err := d.db.ExecContext(ctx, `DELETE FROM metadata WHERE key = ?`, rederivePendingKey); err != nil {
		return fmt.Errorf("clearing the derived columns owed: %w", err)
	}
	d.emitMigrationProgress(rederivePhase, int(maxID), int(maxID))
	slog.Info("derived again the columns of the stored analyses and moves", "rows", repaired)
	return nil
}
