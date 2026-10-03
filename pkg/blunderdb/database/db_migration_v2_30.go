package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// prunedIndexes2_30 are the indexes the 2.30.0 schema no longer declares,
// chosen on the query plans of a real 15.6 M-position library
// (tasks/search-query-plans.txt). EnsureSchema builds indexes by name and
// never drops one, so the step drops them itself.
var prunedIndexes2_30 = []string{
	// decision_type splits the library in two: a filter on it alone ran
	// slower through either index than through the table, and the first page
	// of results paid a sort of every candidate. The dice filter is answered
	// by idx_position_dice, the pip filter by idx_position_pip_diff.
	"idx_position_decision_dice",
	"idx_position_decision_pip",
	// Replaced by the partial idx_position_cube_take: only the take/pass
	// side (is_cube_response = 1) is selective.
	"idx_position_cube_response",
	// Merged into idx_analysis_win_gammon2_covering, which answers the
	// player-2 rate filters from the index alone as the player-1 one does.
	"idx_analysis_win2",
	"idx_analysis_gammon2",
}

// matchDateBackfillKey is the metadata key that says position.match_date
// still has to be derived, and from which position id: set by the 2.30.0
// step, advanced batch by batch, deleted when the pass is over — so an open
// interrupted mid-pass resumes where it stopped.
const matchDateBackfillKey = "backfill_position_match_date"

// migrate_2_29_0_to_2_30_0 is the large-library wave: the provenance columns
// of analysis (engine, depth, creation date), position.match_date, the
// per-file import journal (import_batch_file), player and event aliases, and
// the index pruning above. EnsureSchema adds the columns, tables and new
// indexes after the chain; the two backfills that need them run after it
// (runMigrationChain): position.match_date from matchDateBackfillKey, the
// analysis columns from their NULLs.
func (d *Database) migrate_2_29_0_to_2_30_0(ctx context.Context) error {
	for _, name := range prunedIndexes2_30 {
		if _, err := d.db.ExecContext(ctx, `DROP INDEX IF EXISTS `+name); err != nil {
			return fmt.Errorf("dropping %s: %w", name, err)
		}
	}
	_, err := d.db.ExecContext(ctx,
		`INSERT INTO metadata (key, value) VALUES (?, '0')
		 ON CONFLICT(key) DO UPDATE SET value = '0'`, matchDateBackfillKey)
	return err
}

// backfillBatch is how many rows one backfill transaction writes: large
// enough that the commit is amortised, small enough that an interruption
// loses seconds and the progress callback moves.
const backfillBatch = 20000

// backfillPositionMatchDates derives position.match_date for every position
// while matchDateBackfillKey is set, in id ranges, recording the next id in
// the same transaction as the batch. It reports whether the key was set: the
// key is left in place, for finishLargeLibraryWave to delete once the
// statistics are refreshed.
func (d *Database) backfillPositionMatchDates(ctx context.Context) (bool, error) {
	var from string
	err := d.db.QueryRowContext(ctx, `SELECT value FROM metadata WHERE key = ?`, matchDateBackfillKey).Scan(&from)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	next, err := strconv.ParseInt(from, 10, 64)
	if err != nil {
		next = 0
	}
	var maxID int64
	if err := d.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(id), 0) FROM position`).Scan(&maxID); err != nil {
		return false, err
	}
	for next <= maxID {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		end := next + backfillBatch
		err := d.inTx(ctx, func(tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, `UPDATE position SET match_date =
				(SELECT MIN(m.match_date) FROM move mv
				   JOIN game g ON g.id = mv.game_id
				   JOIN match m ON m.id = g.match_id
				  WHERE mv.position_id = position.id)
				WHERE id >= ? AND id < ?`, next, end); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, `UPDATE metadata SET value = ? WHERE key = ?`,
				strconv.FormatInt(end, 10), matchDateBackfillKey)
			return err
		})
		if err != nil {
			return false, err
		}
		next = end
		d.emitMigrationProgress("position_match_date", int(next), int(maxID))
	}
	return true, nil
}

// finishLargeLibraryWave closes the open crossing 2.30.0, after both
// backfills: a full ANALYZE, then the deletion of matchDateBackfillKey, so an
// interruption before the end redoes the ANALYZE on the next open.
//
// The statistics the file carries describe the 2.29.0 library: they still
// name the pruned indexes' neighbours as they were and say nothing of the new
// ones, and ensureSearchStats keeps any statistics it finds. A sampled
// refresh (PRAGMA analysis_limit, as RefreshSearchStatistics does) does not
// help either: it estimates 1000 rows per key on every index. On a
// 15.6 M-position library both left the score filter (S) and its
// combination with the decision type seven to twenty times slower than with
// a full ANALYZE, which costs about 30 s there against 18 min for the
// backfills. An ANALYZE that fails for another reason than a cancellation is
// logged and not retried: the library is usable, only slower.
func (d *Database) finishLargeLibraryWave(ctx context.Context) error {
	if _, err := d.db.ExecContext(ctx, `ANALYZE`); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		slog.Warn("refreshing the planner statistics after the 2.30.0 migration failed", "err", err)
	}
	_, err := d.db.ExecContext(ctx, `DELETE FROM metadata WHERE key = ?`, matchDateBackfillKey)
	return err
}

// backfillAnalysisProvenance derives analysis_engine, analysis_depth and
// creation_date (engine.AnalysisProvenance) for every analysis row whose
// analysis_engine is NULL: every row on the open crossing 2.30.0, and on
// later opens the few a blob-only writer left. The probe is one step of
// idx_analysis_engine, so an open with nothing to do pays nothing. Resumable
// by construction: a written row is no longer NULL.
func (d *Database) backfillAnalysisProvenance(ctx context.Context) error {
	var probe int
	err := d.db.QueryRowContext(ctx, `SELECT 1 FROM analysis WHERE analysis_engine IS NULL LIMIT 1`).Scan(&probe)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var total int
	if err := d.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM analysis WHERE analysis_engine IS NULL`).Scan(&total); err != nil {
		return err
	}
	done := 0
	var last int64
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		raw, ids, err := d.nullProvenanceBatch(ctx, last)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			break
		}
		decoded, failed := engine.DecodeAnalysesConcurrently(raw)
		err = d.inTx(ctx, func(tx *sql.Tx) error {
			stmt, err := tx.PrepareContext(ctx,
				`UPDATE analysis SET analysis_engine = ?, analysis_depth = ?, creation_date = ? WHERE id = ?`)
			if err != nil {
				return err
			}
			defer stmt.Close()
			for _, id := range ids {
				// An undecodable blob is written as "no entry" so that it is not
				// retried on every open; `blunderdb verify` still reports it.
				eng, depth, created := "", int64(-1), ""
				if a := decoded[id]; a != nil {
					eng, depth, created = engine.AnalysisProvenance(a)
				} else if err := failed[id]; err != nil {
					slog.Warn("analysis provenance: undecodable blob", "analysis_id", id, "error", err)
				}
				var createdVal any
				if created != "" {
					createdVal = created
				}
				if _, err := stmt.ExecContext(ctx, eng, depth, createdVal, id); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		last = ids[len(ids)-1]
		done += len(ids)
		d.emitMigrationProgress("analysis_provenance", done, total)
	}
	if done > 0 {
		slog.Info("derived the provenance of the stored analyses", "analyses", done)
	}
	return nil
}

// nullProvenanceBatch reads the next batch of analysis rows past last whose
// provenance is not derived yet, keyed by analysis id.
func (d *Database) nullProvenanceBatch(ctx context.Context, last int64) (map[int64][]byte, []int64, error) {
	rows, err := d.db.QueryContext(ctx,
		`SELECT id, data FROM analysis WHERE analysis_engine IS NULL AND id > ? ORDER BY id LIMIT ?`,
		last, backfillBatch)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	raw := make(map[int64][]byte, backfillBatch)
	var ids []int64
	for rows.Next() {
		var id int64
		var data []byte
		if err := rows.Scan(&id, &data); err != nil {
			return nil, nil, err
		}
		raw[id] = data
		ids = append(ids, id)
	}
	return raw, ids, rows.Err()
}

// inTx runs fn in a transaction of its own on d.db.
func (d *Database) inTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
