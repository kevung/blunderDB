package sqlite

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Vacuum reclaims disk space left behind by deletions. User-triggered only —
// never at open, its cost is unpredictable. The desktop wrapper, the CLI's
// `vacuum` and the daemon's /ops/maintenance.vacuum all come through here.
//
// Steps, in order:
//
//  1. `PRAGMA wal_checkpoint(TRUNCATE)`, so the "before" size and the
//     free-space check are not fooled by a fat WAL.
//  2. A free-space check refusing the run below roughly twice the file size:
//     VACUUM rebuilds into a fresh file, and failing midway is worse.
//  3. `VACUUM`, as a bare Exec: SQLite refuses it inside a transaction.
//  4. `ANALYZE`, so planner statistics reflect the rebuilt file.
//  5. A second `wal_checkpoint(TRUNCATE)`: under WAL, VACUUM's output goes
//     through the WAL and the file only shrinks once checkpointed.
//
// Before that, compactAnalyses rewrites every analysis blob still in a legacy
// format (raw JSON, zlib, JSON in zstd) as binary at zstd level 7, across all
// cores (engine.RecompressAnalysesConcurrently). Binary blobs are left alone:
// level 19 would save under 2% of their size for hours of CPU on a large
// library. Its errors are logged, not returned: an unreadable row stays in its
// old format, which is better than refusing the compaction.
//
// Returns the file size in bytes before and after; 0 and 0 on ":memory:",
// where VACUUM and ANALYZE still run.
func (s *Storage) Vacuum(ctx context.Context) (storage.VacuumResult, error) {
	if s.sqlDB == nil {
		return storage.VacuumResult{}, fmt.Errorf("vacuum: no database open")
	}

	path, err := mainFilePath(ctx, s.sqlDB)
	if err != nil {
		return storage.VacuumResult{}, fmt.Errorf("vacuum: %w", err)
	}

	if err := s.compactAnalyses(ctx); err != nil {
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
		if mem, known := availableMemoryBytes(); known && mem < vacuumMinMemoryBytes {
			return storage.VacuumResult{}, fmt.Errorf(
				"vacuum: not enough available memory (need at least %s, only %s available): close other applications and retry",
				humanBytes(vacuumMinMemoryBytes), humanBytes(int64(mem)),
			)
		}
		tmp := sqliteTempDir()
		if tmpFree, err := freeSpaceBytes(tmp); err == nil && tmpFree < uint64(sizeBefore) {
			return storage.VacuumResult{}, fmt.Errorf(
				"vacuum: not enough free disk space for the temporary file (need about %s, only %s available in %s): set SQLITE_TMPDIR to a folder with room",
				humanBytes(sizeBefore), humanBytes(int64(tmpFree)), tmp,
			)
		}
		if needed := uint64(sizeBefore) * 2; free < needed {
			return storage.VacuumResult{}, fmt.Errorf(
				"vacuum: not enough free disk space (need about %s, only %s available on the volume holding %s)",
				humanBytes(int64(needed)), humanBytes(int64(free)), path,
			)
		}
	}

	if err := s.vacuumOnFileTemp(ctx); err != nil {
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

// vacuumMinMemoryBytes is the memory VACUUM still needs once its transient
// database lives in a file: the page cache and sort buffers, not the data.
const vacuumMinMemoryBytes = 512 << 20

// sqliteTempDir is where SQLite creates its temporary files: SQLITE_TMPDIR,
// else the system's temporary directory.
func sqliteTempDir() string {
	if d := os.Getenv("SQLITE_TMPDIR"); d != "" {
		return d
	}
	return os.TempDir()
}

// vacuumOnFileTemp runs VACUUM on a dedicated connection whose temp_store is
// FILE. The pool's connections use temp_store=MEMORY, and VACUUM builds its
// transient copy of the whole database in the temp store: in RAM that is an
// out-of-memory kill on a database larger than the machine. The temp file
// goes where SQLite puts them (SQLITE_TMPDIR, else TMPDIR, else /tmp): the
// directory is process-wide state this code must not touch, since the daemon
// shares the process with other connections. On a small /tmp, point
// SQLITE_TMPDIR at the database's volume. The setting is undone before the connection returns to the pool. VACUUM
// cannot run inside a transaction: bare Exec, not withTx.
func (s *Storage) vacuumOnFileTemp(ctx context.Context) error {
	conn, err := s.sqlDB.Conn(ctx)
	if err != nil {
		return err
	}
	// Close hands the connection back to the pool: undo the setting first.
	// Not ctx: a cancelled vacuum must still restore the pooled connection.
	defer func() {
		_, _ = conn.ExecContext(context.WithoutCancel(ctx), `PRAGMA temp_store=MEMORY`)
		conn.Close()
	}()

	if _, err := conn.ExecContext(ctx, `PRAGMA temp_store=FILE`); err != nil {
		return fmt.Errorf("temp_store: %w", err)
	}
	_, err = conn.ExecContext(ctx, `VACUUM`)
	return err
}

// compactAnalysesBatchSize is how many analysis rows are read and,
// if needed, rewritten per transaction — small enough that a big table does
// not hold one giant transaction open for the whole pass.
const compactAnalysesBatchSize = 2000

type analysisRow struct {
	id   int64
	data []byte
}

// fetchAnalysisBatch reads one page of analysis.data ordered by id,
// closing its cursor before returning so the caller is free to write on the
// same connection right after.
func fetchAnalysisBatch(ctx context.Context, db execer, afterID int64, limit int) ([]analysisRow, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, data FROM analysis WHERE id > ? ORDER BY id LIMIT ?`, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var batch []analysisRow
	for rows.Next() {
		var r analysisRow
		if err := rows.Scan(&r.id, &r.data); err != nil {
			return nil, err
		}
		batch = append(batch, r)
	}
	return batch, rows.Err()
}

// compactAnalyses walks analysis.data in id order and rewrites any row not
// already binary. engine.NeedsRecompression is a cheap header check, so a
// database with nothing legacy costs one full-table SELECT and no writes.
func (s *Storage) compactAnalyses(ctx context.Context) error {
	var lastID int64
	var scanned, upgraded int
	for {
		batch, err := fetchAnalysisBatch(ctx, s.sqlDB, lastID, compactAnalysesBatchSize)
		if err != nil {
			return err
		}
		if len(batch) == 0 {
			break
		}
		lastID = batch[len(batch)-1].id
		scanned += len(batch)

		var todo []analysisRow
		var blobs [][]byte
		for _, r := range batch {
			if engine.NeedsRecompression(r.data) {
				todo = append(todo, r)
				blobs = append(blobs, r.data)
			}
		}
		if len(todo) == 0 {
			continue
		}
		freshBlobs, encErrs := engine.RecompressAnalysesConcurrently(blobs)

		err = withTx(ctx, s.sqlDB, func(tx execer) error {
			for i, r := range todo {
				fresh, err := freshBlobs[i], encErrs[i]
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
		slog.Info("vacuum: re-encoded analysis blobs", "scanned", scanned, "upgraded", upgraded)
	}
	return nil
}

// DatabaseSizeBytes reports the current size of the SQLite main file in
// bytes, for the daemon's blunderdb_database_size_bytes gauge: a plain
// os.Stat without checkpoint, good enough for an unattended gauge. Returns
// 0, nil on ":memory:".
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

// ErrVacuumIntoNoRoom is VacuumInto refusing a copy the volume of its target
// cannot hold.
var ErrVacuumIntoNoRoom = errors.New("not enough free disk space for the compacted copy")

// VacuumInto is the first half of a vacuum by file replacement: the same
// compaction of legacy analyses as Vacuum, then `VACUUM INTO target`, which
// writes the compacted database straight into a new file — no transient
// copy in the temp store, no WAL the size of the database. The peak is the
// file plus its compacted copy, against about three times the file for an
// in-place VACUUM under WAL. Swapping target in is the owner's business: only
// the holder of the file can close every connection to it first
// (database.Vacuum).
//
// target must not exist. Returns the size of the file before, WAL folded in.
func (s *Storage) VacuumInto(ctx context.Context, target string) (int64, error) {
	if s.sqlDB == nil {
		return 0, fmt.Errorf("vacuum: no database open")
	}
	path, err := mainFilePath(ctx, s.sqlDB)
	if err != nil {
		return 0, fmt.Errorf("vacuum: %w", err)
	}
	if path == "" {
		return 0, fmt.Errorf("vacuum into: the database has no file")
	}
	if err := s.compactAnalyses(ctx); err != nil {
		return 0, fmt.Errorf("vacuum: recompress analyses: %w", err)
	}
	if _, err := s.sqlDB.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return 0, fmt.Errorf("vacuum: wal checkpoint: %w", err)
	}
	sizeBefore, err := fileSize(path)
	if err != nil {
		return 0, fmt.Errorf("vacuum: %w", err)
	}
	free, err := freeSpaceBytes(filepath.Dir(target))
	if err != nil {
		return sizeBefore, fmt.Errorf("vacuum: could not determine free disk space: %w", err)
	}
	if free < uint64(sizeBefore) {
		return sizeBefore, fmt.Errorf("vacuum: %w (need about %s, only %s available beside %s)",
			ErrVacuumIntoNoRoom, humanBytes(sizeBefore), humanBytes(int64(free)), path)
	}
	if _, err := s.sqlDB.ExecContext(ctx, `VACUUM INTO ?`, target); err != nil {
		_ = os.Remove(target)
		return sizeBefore, fmt.Errorf("vacuum into: %w", err)
	}
	// The copy is created under the umask; it replaces the library, so it
	// takes the library's permissions (a 0600 file must not come back 0644).
	fi, err := os.Stat(path)
	if err == nil {
		err = os.Chmod(target, fi.Mode().Perm())
	}
	if err != nil {
		_ = os.Remove(target)
		return sizeBefore, fmt.Errorf("vacuum into: keeping the file mode: %w", err)
	}
	return sizeBefore, nil
}

// FilePath is the file the main database lives in, "" for ":memory:".
func (s *Storage) FilePath(ctx context.Context) (string, error) {
	if s.sqlDB == nil {
		return "", fmt.Errorf("no database open")
	}
	return mainFilePath(ctx, s.sqlDB)
}
