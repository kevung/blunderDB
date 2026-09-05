package sqlite

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Vacuum reclaims disk space left behind by deletions (matches, tournaments,
// purges) that SQLite never shrinks the file for on its own. It is an
// explicit, user-triggered action only — never run automatically at open,
// since its cost is unpredictable on a large database. The desktop wrapper
// (database.Database.Vacuum), the CLI's `vacuum` and the daemon's
// /ops/maintenance.vacuum all come through here.
//
// Steps, in order:
//
//  1. `PRAGMA wal_checkpoint(TRUNCATE)` folds the WAL back into the main
//     file first, so the "before" size (and the free-space check below)
//     reflect the database honestly rather than a stale main file sitting
//     next to a fat WAL.
//  2. A free-space check refuses the run outright when the volume holding
//     the file has less than roughly twice its size available. SQLite's
//     VACUUM rebuilds the whole database into a fresh file before swapping
//     it in, so it transiently needs about that much headroom; failing
//     partway through a rebuild is far worse than refusing up front.
//  3. `VACUUM` itself. SQLite refuses to run it inside a transaction, so
//     it is issued as a bare Exec rather than through any transaction
//     helper.
//  4. `ANALYZE`, so the query planner's statistics reflect the rebuilt
//     file instead of the pre-vacuum layout (fiche 05's synergy).
//  5. A second `wal_checkpoint(TRUNCATE)`. Under WAL journal mode (the
//     mode this package always opens in), VACUUM's rebuilt content is
//     itself written through the WAL rather than truncating the main file
//     in place — the file on disk does not actually shrink until that gets
//     checkpointed back. Skipping this step would report a false "nothing
//     reclaimed" even though the rebuild succeeded.
//
// Before any of that, recompressLegacyAnalyses (#180) walks the analysis
// table and upgrades any row still holding the pre-zstd codec (raw JSON or
// zlib — see engine.RecompressAnalysisData) to the current zstd+dictionary
// format. Vacuum already rewrites the whole file and already asks the user
// to accept an unpredictable cost, which makes it the natural trigger for a
// pass that is otherwise easy to never get around to: a database opened
// only with today's binary would otherwise carry its original-import codec
// forever. The upgrade errs are non-fatal (logged, not returned) — a handful
// of rows this pass could not read stay in their old format and are picked
// up by ordinary reads/writes or the next vacuum; refusing the whole
// compaction over that would be a worse outcome for the user than a few
// bytes not yet reclaimed.
//
// Returns the file size in bytes before and after. On an in-memory database
// (tests, `:memory:`) there is no file to size or free-space-check against;
// VACUUM and ANALYZE still run, and both sizes are reported as 0.
func (s *Storage) Vacuum(ctx context.Context) (storage.VacuumResult, error) {
	if s.sqlDB == nil {
		return storage.VacuumResult{}, fmt.Errorf("vacuum: no database open")
	}

	path, err := mainFilePath(ctx, s.sqlDB)
	if err != nil {
		return storage.VacuumResult{}, fmt.Errorf("vacuum: %w", err)
	}

	if err := s.recompressLegacyAnalyses(ctx); err != nil {
		return storage.VacuumResult{}, fmt.Errorf("vacuum: recompress analyses: %w", err)
	}

	// Fold the WAL back into the main file before sizing or checking free
	// space: otherwise "before" undercounts work the VACUUM still has to do.
	if _, err := s.sqlDB.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return storage.VacuumResult{}, fmt.Errorf("vacuum: wal checkpoint: %w", err)
	}

	var sizeBefore int64
	if path != "" {
		sizeBefore, err = fileSize(path)
		if err != nil {
			return storage.VacuumResult{}, fmt.Errorf("vacuum: %w", err)
		}

		free, spaceErr := freeSpaceBytes(path)
		if spaceErr != nil {
			// Cannot verify there is enough room: refuse rather than risk
			// VACUUM failing midway through rebuilding the file.
			return storage.VacuumResult{}, fmt.Errorf("vacuum: could not determine free disk space: %w", spaceErr)
		}
		if needed := uint64(sizeBefore) * 2; free < needed {
			return storage.VacuumResult{}, fmt.Errorf(
				"vacuum: not enough free disk space (need about %s, only %s available on the volume holding %s)",
				humanBytes(int64(needed)), humanBytes(int64(free)), path,
			)
		}
	}

	// VACUUM cannot run inside a transaction; this is a bare Exec on the
	// pool, deliberately not going through withTx.
	if _, err := s.sqlDB.ExecContext(ctx, `VACUUM`); err != nil {
		return storage.VacuumResult{SizeBefore: sizeBefore}, fmt.Errorf("vacuum: %w", err)
	}

	if _, err := s.sqlDB.ExecContext(ctx, `ANALYZE`); err != nil {
		return storage.VacuumResult{SizeBefore: sizeBefore}, fmt.Errorf("vacuum: analyze: %w", err)
	}

	// See step 5 in the doc comment: VACUUM's rebuild goes through the WAL
	// under this package's journal mode, so the file only shrinks once that
	// is checkpointed back.
	if _, err := s.sqlDB.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return storage.VacuumResult{SizeBefore: sizeBefore}, fmt.Errorf("vacuum: wal checkpoint after vacuum: %w", err)
	}

	var sizeAfter int64
	if path != "" {
		sizeAfter, err = fileSize(path)
		if err != nil {
			return storage.VacuumResult{SizeBefore: sizeBefore}, fmt.Errorf("vacuum: %w", err)
		}
	}

	return storage.VacuumResult{SizeBefore: sizeBefore, SizeAfter: sizeAfter}, nil
}

// recompressLegacyAnalysesBatchSize is how many analysis rows are read and,
// if needed, rewritten per transaction — small enough that a big table does
// not hold one giant transaction open for the whole pass (within the report's
// suggested 1,000-5,000 range, see docs/recherche/P11-compression-blobs.md).
const recompressLegacyAnalysesBatchSize = 2000

type legacyAnalysisRow struct {
	id   int64
	data []byte
}

// fetchLegacyAnalysisBatch reads one page of analysis.data ordered by id,
// closing its cursor before returning so the caller is free to write on the
// same connection right after.
func fetchLegacyAnalysisBatch(ctx context.Context, db execer, afterID int64, limit int) ([]legacyAnalysisRow, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, data FROM analysis WHERE id > ? ORDER BY id LIMIT ?`, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var batch []legacyAnalysisRow
	for rows.Next() {
		var r legacyAnalysisRow
		if err := rows.Scan(&r.id, &r.data); err != nil {
			return nil, err
		}
		batch = append(batch, r)
	}
	return batch, rows.Err()
}

// recompressLegacyAnalyses walks analysis.data in id order and rewrites any
// row not already in the current zstd format. engine.NeedsRecompression is a
// cheap prefix check, so a database that has already been through one vacuum
// (or was created after #180) costs one full-table SELECT of already-current
// rows and no writes at all.
func (s *Storage) recompressLegacyAnalyses(ctx context.Context) error {
	var lastID int64
	var scanned, upgraded int
	for {
		batch, err := fetchLegacyAnalysisBatch(ctx, s.sqlDB, lastID, recompressLegacyAnalysesBatchSize)
		if err != nil {
			return err
		}
		if len(batch) == 0 {
			break
		}
		lastID = batch[len(batch)-1].id
		scanned += len(batch)

		err = withTx(ctx, s.sqlDB, func(tx execer) error {
			for _, r := range batch {
				if !engine.NeedsRecompression(r.data) {
					continue
				}
				fresh, err := engine.RecompressAnalysisData(r.data)
				if err != nil {
					// A row this pass cannot read is left exactly as it was:
					// still readable by DecompressAnalysisData's fallback
					// paths, just not upgraded this time.
					slog.Warn("vacuum: skipping unreadable analysis row", "id", r.id, "error", err)
					continue
				}
				if _, err := tx.ExecContext(ctx,
					`UPDATE analysis SET data = ? WHERE id = ?`, fresh, r.id); err != nil {
					return fmt.Errorf("id %d: %w", r.id, err)
				}
				upgraded++
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	if upgraded > 0 {
		slog.Info("vacuum: recompressed legacy analysis blobs", "scanned", scanned, "upgraded", upgraded)
	}
	return nil
}

// DatabaseSizeBytes reports the current size of the SQLite main file in
// bytes, for the daemon's blunderdb_database_size_bytes gauge (#238; see
// server.sizeProvider) — a plain os.Stat, not the more careful
// wal_checkpoint-then-stat Vacuum does, since this runs on an unattended
// timer rather than a user-triggered action and a WAL-inclusive
// approximation is good enough for a gauge. Returns 0, nil on an in-memory
// database (tests, `:memory:`), which has no file to size.
func (s *Storage) DatabaseSizeBytes(ctx context.Context) (int64, error) {
	if s.sqlDB == nil {
		return 0, fmt.Errorf("database size: no database open")
	}
	path, err := mainFilePath(ctx, s.sqlDB)
	if err != nil {
		return 0, fmt.Errorf("database size: %w", err)
	}
	if path == "" {
		return 0, nil
	}
	return fileSize(path)
}

// mainFilePath returns the absolute path SQLite has the "main" database open
// against, or "" for an in-memory database.
func mainFilePath(ctx context.Context, db execer) (string, error) {
	rows, err := db.QueryContext(ctx, `PRAGMA database_list`)
	if err != nil {
		return "", fmt.Errorf("database_list: %w", err)
	}
	defer rows.Close()

	var seq int
	var name, file string
	for rows.Next() {
		if err := rows.Scan(&seq, &name, &file); err != nil {
			return "", fmt.Errorf("database_list: %w", err)
		}
		if name == "main" {
			return file, nil
		}
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("database_list: %w", err)
	}
	return "", nil
}

// fileSize is os.Stat().Size(), wrapped so callers get a %w-wrappable error.
func fileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("stat %s: %w", path, err)
	}
	return info.Size(), nil
}

// humanBytes formats a byte count for the free-space error message
// (e.g. "12.3 MiB"). Kept local rather than shared with the CLI's own copy:
// this one only ever renders into an error string, not a report table.
func humanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMG"[exp])
}
