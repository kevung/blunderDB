package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// VacuumResult reports the file-size effect of a Vacuum call, in bytes. A
// struct because Wails v2 binds at most (value, error): a third return value
// reaches JS as nil. Not an alias of storage.VacuumResult, so the generated
// Wails model keeps its namespace.
type VacuumResult struct {
	SizeBefore int64
	SizeAfter  int64
	// TrashPurged is how many trash entries the run dropped for being older
	// than domain.TrashRetentionDays (ADR-0036). The trash is purged HERE and
	// nowhere else, never on open.
	TrashPurged int
}

// Vacuum reclaims disk space left behind by deletions (matches, tournaments,
// purges) that SQLite never shrinks the file for on its own. It is an
// explicit, user-triggered action only — never run automatically at open,
// since its cost is unpredictable on a large database.
//
// The compaction lives in sqlite.Storage, shared by every mode. The desktop
// and the CLI own their file, so they vacuum by replacing it
// (vacuumBySwap), which needs half the free space; the in-place VACUUM of
// sqlite.Storage.Vacuum — the daemon's, which shares its pool — is the
// fallback when the file cannot be replaced. This wrapper takes d.mu
// exclusively: VACUUM rewrites the whole file and must not overlap a reader.
func (d *Database) Vacuum() (VacuumResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return VacuumResult{}, fmt.Errorf("vacuum: no database open")
	}
	ctx := context.Background()
	// Purge before compacting, so the space the purge frees is space this
	// VACUUM reclaims rather than space the next one will.
	purged, err := d.store.Trash().Purge(ctx, "", domain.TrashRetentionDays)
	if err != nil {
		return VacuumResult{}, err
	}
	if !d.readOnly {
		before, after, err := d.vacuumBySwap(ctx)
		switch {
		case err == nil:
			return VacuumResult{SizeBefore: before, SizeAfter: after, TrashPurged: purged}, nil
		case !errors.Is(err, errVacuumSwapUnsafe):
			return VacuumResult{SizeBefore: before, TrashPurged: purged}, err
		}
		slog.Info("vacuum: another connection holds the file; vacuuming in place")
	}
	res, err := d.store.Vacuum(ctx)
	return VacuumResult{SizeBefore: res.SizeBefore, SizeAfter: res.SizeAfter, TrashPurged: purged}, err
}

// errVacuumSwapUnsafe is vacuumBySwap declining: the file cannot be replaced
// right now, the in-place VACUUM still can run.
var errVacuumSwapUnsafe = errors.New("vacuum: the file cannot be replaced")

// vacuumBySwap vacuums by file replacement (sqlite.Storage.VacuumInto): the
// compacted copy is written beside the file, every connection of this
// instance is closed, and the copy is renamed over the file. It needs the
// file's size free instead of twice it, which is what lets a 14 GB library be
// vacuumed on a disk with 15 GB left.
//
// The file is replaced only once nothing else has it open. The last
// connection to close a WAL database checkpoints and deletes its -wal file,
// so a -wal still present after this instance closed its own means another
// process (a read-only instance, a sqlite3 shell) is connected: the copy is
// dropped and errVacuumSwapUnsafe returned. A failure after the rename point
// leaves the compacted file in place, which is the whole library.
func (d *Database) vacuumBySwap(ctx context.Context) (before, after int64, err error) {
	path, err := d.store.FilePath(ctx)
	if err != nil || path == "" {
		return 0, 0, errVacuumSwapUnsafe
	}
	tmp := path + ".vacuum"
	if err := os.Remove(tmp); err != nil && !errors.Is(err, os.ErrNotExist) {
		return 0, 0, fmt.Errorf("vacuum: removing a leftover copy: %w", err)
	}
	before, err = d.store.VacuumInto(ctx, tmp)
	if err != nil {
		return before, 0, err
	}
	if err := syncFile(tmp); err != nil {
		_ = os.Remove(tmp)
		return before, 0, fmt.Errorf("vacuum: %w", err)
	}

	_, _ = d.db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`)
	if err := d.db.Close(); err != nil {
		slog.Warn("vacuum: closing the database before the swap", "err", err)
	}
	d.db = nil
	swapped := false
	if _, statErr := os.Stat(path + "-wal"); errors.Is(statErr, os.ErrNotExist) {
		if err := os.Rename(tmp, path); err == nil {
			swapped = true
			_ = os.Remove(path + "-shm")
			syncDir(filepath.Dir(path))
		} else {
			slog.Warn("vacuum: replacing the file failed; vacuuming in place", "err", err)
		}
	}
	if !swapped {
		_ = os.Remove(tmp)
	}
	if err := d.reopenAfterSwap(path); err != nil {
		return before, 0, fmt.Errorf("vacuum: reopening the database: %w", err)
	}
	if !swapped {
		return before, 0, errVacuumSwapUnsafe
	}
	if _, err := d.db.ExecContext(ctx, `ANALYZE`); err != nil {
		return before, 0, fmt.Errorf("vacuum: analyze: %w", err)
	}
	if _, err := d.db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return before, 0, fmt.Errorf("vacuum: wal checkpoint after vacuum: %w", err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		return before, 0, fmt.Errorf("vacuum: %w", err)
	}
	return before, fi.Size(), nil
}

// reopenAfterSwap opens path again as OpenDatabase does, under the d.mu the
// caller holds. The schema is the one the closed handle had, so there is no
// chain to run.
func (d *Database) reopenAfterSwap(path string) error {
	db, err := sql.Open("sqlite", sqlite.DSN(path))
	if err != nil {
		return err
	}
	sqlite.ConfigurePool(db, path)
	if err := db.Ping(); err != nil {
		db.Close()
		return err
	}
	d.db = db
	d.rebuildStore()
	d.rebindServices()
	return nil
}

// rebindServices moves the cached services onto the new store. The swapped
// file holds the same rows under the same ids and revisions, so the drafts
// being typed keep their undo stacks and an open Duel stays open; the d.mu
// the caller holds for writing excludes every call reading their store.
func (d *Database) rebindServices() {
	d.transcriptMu.Lock()
	if d.transcriptSvc != nil {
		d.transcriptSvc.Rebind(d.store)
		d.transcriptOn = d.store
	}
	d.transcriptMu.Unlock()
	d.duelMu.Lock()
	if d.duelSvc != nil {
		d.duelSvc.Rebind(d.store)
		d.duelOn = d.store
	}
	d.duelMu.Unlock()
}

func syncFile(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// syncDir makes the rename durable where the platform allows a directory to
// be synced; elsewhere it is a no-op.
func syncDir(dir string) {
	if f, err := os.Open(dir); err == nil {
		_ = f.Sync()
		f.Close()
	}
}
