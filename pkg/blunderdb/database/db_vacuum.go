package database

import (
	"fmt"
	"os"
)

// VacuumResult reports the file-size effect of a Vacuum call, in bytes. It
// exists (rather than Vacuum returning two int64s) because Wails v2's method
// binding only understands a bound method returning exactly (value, error)
// or a single value — a third return value is silently dropped at the JS
// boundary, not just left untyped, so a (before, after int64, error) shape
// would work from Go/CLI callers but hand the GUI button back nil, nil
// unconditionally. See IndividualSaveResult for the same reason repeated
// elsewhere in this package.
type VacuumResult struct {
	SizeBefore int64
	SizeAfter  int64
}

// Vacuum reclaims disk space left behind by deletions (matches, tournaments,
// purges) that SQLite never shrinks the file for on its own. It is an
// explicit, user-triggered action only — never run automatically at open,
// since its cost is unpredictable on a large database.
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
//     it is issued as a bare Exec on the shared connection rather than
//     through any transaction helper.
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
func (d *Database) Vacuum() (VacuumResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return VacuumResult{}, fmt.Errorf("vacuum: no database open")
	}

	path, err := d.mainFilePathLocked()
	if err != nil {
		return VacuumResult{}, fmt.Errorf("vacuum: %w", err)
	}

	// Fold the WAL back into the main file before sizing or checking free
	// space: otherwise "before" undercounts work the VACUUM still has to do.
	if _, err := d.db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return VacuumResult{}, fmt.Errorf("vacuum: wal checkpoint: %w", err)
	}

	var sizeBefore int64
	if path != "" {
		sizeBefore, err = vacuumFileSize(path)
		if err != nil {
			return VacuumResult{}, fmt.Errorf("vacuum: %w", err)
		}

		free, spaceErr := freeSpaceBytes(path)
		if spaceErr != nil {
			// Cannot verify there is enough room: refuse rather than risk
			// VACUUM failing midway through rebuilding the file.
			return VacuumResult{}, fmt.Errorf("vacuum: could not determine free disk space: %w", spaceErr)
		}
		if needed := uint64(sizeBefore) * 2; free < needed {
			return VacuumResult{}, fmt.Errorf(
				"vacuum: not enough free disk space (need about %s, only %s available on the volume holding %s)",
				vacuumHumanBytes(int64(needed)), vacuumHumanBytes(int64(free)), path,
			)
		}
	}

	// VACUUM cannot run inside a transaction; this is a bare Exec on the
	// shared connection, deliberately not going through a Begin/Commit helper.
	if _, err := d.db.Exec(`VACUUM`); err != nil {
		return VacuumResult{SizeBefore: sizeBefore}, fmt.Errorf("vacuum: %w", err)
	}

	if _, err := d.db.Exec(`ANALYZE`); err != nil {
		return VacuumResult{SizeBefore: sizeBefore}, fmt.Errorf("vacuum: analyze: %w", err)
	}

	// See step 5 in the doc comment: VACUUM's rebuild goes through the WAL
	// under this package's journal mode, so the file only shrinks once that
	// is checkpointed back.
	if _, err := d.db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return VacuumResult{SizeBefore: sizeBefore}, fmt.Errorf("vacuum: wal checkpoint after vacuum: %w", err)
	}

	var sizeAfter int64
	if path != "" {
		sizeAfter, err = vacuumFileSize(path)
		if err != nil {
			return VacuumResult{SizeBefore: sizeBefore}, fmt.Errorf("vacuum: %w", err)
		}
	}

	return VacuumResult{SizeBefore: sizeBefore, SizeAfter: sizeAfter}, nil
}

// mainFilePathLocked returns the absolute path SQLite has the "main" database
// open against, or "" for an in-memory database. The caller must hold d.mu.
func (d *Database) mainFilePathLocked() (string, error) {
	rows, err := d.db.Query(`PRAGMA database_list`)
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

// vacuumFileSize is os.Stat().Size(), wrapped so callers get a %w-wrappable error.
func vacuumFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("stat %s: %w", path, err)
	}
	return info.Size(), nil
}

// vacuumHumanBytes formats a byte count for the free-space error message
// (e.g. "12.3 MiB"). Kept local rather than shared with the CLI's own copy:
// this one only ever renders into an error string, not a report table.
func vacuumHumanBytes(b int64) string {
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
