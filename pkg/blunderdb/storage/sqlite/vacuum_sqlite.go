package sqlite

import (
	"context"
	"fmt"
	"os"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Vacuum reclaims disk space left behind by deletions (matches, tournaments,
// purges) that SQLite never shrinks the file for on its own. It is an
// explicit, user-triggered action only — never run automatically at open,
// since its cost is unpredictable on a large database. The desktop wrapper
// (database.Database.Vacuum), the CLI's `vacuum` and the daemon's
// /v1/maintenance.vacuum all come through here.
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
