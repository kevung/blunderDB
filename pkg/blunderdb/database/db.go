package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/adrg/xdg"

	_ "modernc.org/sqlite"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

type Database struct {
	db                *sql.DB
	mu                sync.RWMutex                        // RWMutex allows concurrent reads
	cancelMu          sync.Mutex                          // guards importCancel (held briefly, never with mu)
	importCancel      context.CancelFunc                  // cancels the in-flight import/migration; nil when idle
	migrationProgress func(phase string, done, total int) // optional progress callback (GUI only)
	store             *sqlite.Storage                     // SQLite Storage backend, wraps db (P2)
	lock              *fileLock                           // single-writer advisory lock on the open file (nil for :memory:/read-only)
	readOnly          bool                                // opened read-only because another instance holds the write lock
}

// acquireFileLock takes the single-writer advisory lock for a file-backed
// database before it is opened. On success d.lock holds it and d.readOnly is
// false. If another instance already holds it, d.readOnly is set true and the
// caller must open the file read-only (ADR-0004: multiple instances are allowed,
// but a database may not be opened read-write twice). A lock-infrastructure
// failure (e.g. a read-only directory) is NON-fatal: single-instance is an
// optional capability that must never block opening, so d.readOnly stays false
// and the open proceeds unguarded. :memory: and the empty path are never locked.
// lockPathFor returns the file whose advisory lock guards a database against a second
// writer.
//
// It lives in the cache directory rather than beside the database. Two reasons, and the
// second is the one that matters: a stray `cours.db.lock` next to someone's database reads
// as debris, and it cannot simply be deleted when the lock is released — another instance
// may already hold a descriptor to that inode, so unlinking it would let a third instance
// create a fresh file and take a lock that excludes nobody. Keeping the marker out of sight
// avoids the clutter without touching the correctness of the lock.
//
// The name is derived from the database's absolute path, so the same database always maps
// to the same lock wherever it is opened from. If the cache directory cannot be created —
// single-instance locking is an optional capability (ADR-0004) — it falls back beside the
// database, which is where it always used to be.
func lockPathFor(dbPath string) string {
	abs, err := filepath.Abs(dbPath)
	if err != nil {
		abs = dbPath
	}
	dir := filepath.Join(xdg.CacheHome, "blunderDB", "locks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		slog.Debug("lock directory unavailable; locking beside the database", "err", err)
		return dbPath + ".lock"
	}
	sum := sha256.Sum256([]byte(abs))
	return filepath.Join(dir, hex.EncodeToString(sum[:16])+".lock")
}

func (d *Database) acquireFileLock(path string) {
	d.releaseFileLock()
	d.readOnly = false
	if path == "" || path == ":memory:" {
		return
	}
	lock, ok, err := tryLockExclusive(lockPathFor(path))
	if err != nil {
		slog.Warn("single-instance lock unavailable; opening without it", "path", path, "err", err)
		return
	}
	if !ok {
		slog.Info("database already open in another instance; opening read-only", "path", path)
		d.readOnly = true
		return
	}
	d.lock = lock
}

// releaseFileLock drops the single-writer lock if held.
func (d *Database) releaseFileLock() {
	if d.lock != nil {
		if err := d.lock.release(); err != nil {
			slog.Warn("releasing single-instance lock failed", "err", err)
		}
		d.lock = nil
	}
}

// IsReadOnly reports whether the current database was opened read-only because
// another instance holds the write lock. The GUI surfaces this as a notice.
func (d *Database) IsReadOnly() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.readOnly
}

// beginCancellableImport creates a fresh cancellable context for an import or
// migration and registers its cancel func so CancelImport can abort it from
// another goroutine (e.g. the Wails frontend) while the operation holds d.mu.
// cancelMu — never d.mu — guards the registration, so CancelImport never blocks
// on the running import. The returned done func must be deferred: it clears the
// registration and releases the context's resources.
func (d *Database) beginCancellableImport() (context.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	d.cancelMu.Lock()
	d.importCancel = cancel
	d.cancelMu.Unlock()
	return ctx, func() {
		d.cancelMu.Lock()
		d.importCancel = nil
		d.cancelMu.Unlock()
		cancel()
	}
}

// CancelImport aborts any in-flight import or migration started through the
// Database wrapper. It is bound to the Wails frontend. No-op when idle.
func (d *Database) CancelImport() {
	d.cancelMu.Lock()
	cancel := d.importCancel
	d.cancelMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// rebuildStore (re)creates the SQLite Storage that wraps the current *sql.DB.
// It must be called after SetupDatabase/OpenDatabase replace d.db. The Storage
// borrows the handle (sqlite.New): d.db stays owned by this Database.
func (d *Database) rebuildStore() {
	d.store = sqlite.New(d.db)
}

func NewDatabase() *Database {
	return &Database{}
}

// conn returns the underlying *sql.DB handle. It is deliberately unexported:
// *Database is bound wholesale to the Wails frontend (main.go passes it in
// extraBinds), so an exported method here becomes a JS-callable binding that
// hands the raw handle straight to the frontend — never a mode's feature
// (B.8, #176). Code within this package calls it directly; a caller outside
// the package (scripts/demodb, tests in other packages) goes through RawConn.
// It may be nil before Setup/Open.
func (d *Database) conn() *sql.DB {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db
}

// RawConn returns d's underlying *sql.DB handle, for maintenance scripts and
// tests that need direct SQL access outside the Storage contract (e.g.
// scripts/demodb, which builds the embedded demo database with queries
// Database has no method for). It is a package-level function rather than a
// method so that binding *Database to the Wails frontend never exposes it to
// the GUI — see the unexported conn() above. Never call this from
// GUI-reachable code; add a named Database method instead (see Checkpoint).
func RawConn(d *Database) *sql.DB {
	return d.conn()
}

// Checkpoint truncates the write-ahead log into the main database file
// (PRAGMA wal_checkpoint(TRUNCATE)). The CLI's batch importer calls this
// after every successfully imported match to keep the WAL file bounded
// during a long run, rather than reaching for Conn().Exec directly at the
// call site (which also read d.db without going through the lock every other
// writer takes). Best-effort: a checkpoint that cannot complete (e.g. a
// concurrent reader holding an older snapshot open) is not a reason to fail
// the import, so the error is returned for the caller to log rather than
// treated as fatal.
func (d *Database) Checkpoint() error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return nil
	}
	_, err := d.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	return err
}

// Close closes the underlying connection and clears it. It is safe to call
// when the connection is already nil or closed.
func (d *Database) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.releaseFileLock()
	wasReadOnly := d.readOnly
	d.readOnly = false
	if d.db == nil {
		return nil
	}
	// PRAGMA optimize (SQLite docs: run before closing every long-lived
	// connection) mirrors the standalone Storage backend's Close (storage/
	// sqlite/sqlite.go, D9/fiche-05 T7) — this wrapper opens its own *sql.DB
	// rather than going through Storage.Close, so it needs its own call.
	// Best-effort and skipped on a read-only handle (query_only=ON rejects the
	// write ANALYZE may attempt internally; nothing useful to optimize there
	// anyway).
	if !wasReadOnly {
		_, _ = d.db.Exec(`PRAGMA optimize`)
	}
	err := d.db.Close()
	d.db = nil
	return err
}

func (d *Database) SetupDatabase(path string) (err error) {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	// Close the currently opened database, if any. Best-effort like the
	// analogous close in Close() itself: the handle is discarded and replaced
	// right below regardless, but a failure here (e.g. a pooled connection
	// that would not release) is worth a log line rather than silence (B.13,
	// #181 — this used to swallow the error outright).
	if d.db != nil {
		if err := d.db.Close(); err != nil {
			slog.Warn("closing the previously open database failed", "err", err)
		}
	}

	// Creating/erasing a database requires the write lock. If another instance
	// holds it we cannot wipe the file underneath it — refuse rather than open
	// read-only (there is nothing to create read-only).
	d.acquireFileLock(path)
	if d.readOnly {
		d.readOnly = false
		return fmt.Errorf("database %q is open in another instance; close it before creating/replacing it", path)
	}

	// From here on, neither the file lock nor an opened *sql.DB handle may
	// leak on a mid-setup failure: any pragma/table-creation error below used
	// to return with the lock still held and d.db still set, wedging the
	// wrapper (a later Setup/Open could never re-acquire the lock, and the
	// leaked handle was never closed). Roll back to a clean "never opened"
	// state on any error, named-return style so every existing `return err`
	// below is covered without threading cleanup through each one.
	defer func() {
		if err != nil {
			if d.db != nil {
				d.db.Close()
				d.db = nil
			}
			d.releaseFileLock()
		}
	}()

	// The PRAGMAs (foreign_keys=ON, busy_timeout, WAL, …) travel in the DSN
	// so the driver replays them on EVERY pooled connection. A PRAGMA run
	// after sql.Open configures only the one connection it lands on; with a
	// ten-connection pool the other nine ran with foreign_keys=OFF, so a
	// DeleteMatch served by one of them skipped ON DELETE CASCADE and left
	// game/move/move_analysis orphans (issue #157; blunderdb verify counts
	// them).
	d.db, err = sql.Open("sqlite", sqlite.DSN(path))
	if err != nil {
		return err
	}

	// Size the connection pool. Critical for ":memory:": each pooled
	// connection is a SEPARATE empty in-memory database, so concurrent reads
	// (RWMutex allows multiple readers) would otherwise hit a fresh connection
	// with no schema -> "no such table". ConfigurePool pins ":memory:" to a
	// single connection; file-backed DBs are allowed to grow.
	sqlite.ConfigurePool(d.db, path)

	// Erase any content in the database
	_, err = d.db.Exec(`
		PRAGMA writable_schema = 1;
		DELETE FROM sqlite_master WHERE type IN ('table', 'index', 'trigger');
		PRAGMA writable_schema = 0;
		VACUUM;
		PRAGMA INTEGRITY_CHECK;
	`)
	if err != nil {
		return err
	}

	// The schema itself lives in one place, storage/sqlite's Bootstrap
	// (schema_sqlite.go): the DDL and the database_version row it writes are
	// exactly what the headless daemon bootstraps, so a database created here
	// (GUI "new database", CLI create, :memory:) cannot drift from one
	// created by sqlite.Open. schema_parity_test.go holds the two to it.
	if err = sqlite.Bootstrap(context.Background(), d.db); err != nil {
		return err
	}

	d.rebuildStore()
	return nil
}

func (d *Database) OpenDatabase(path string) (err error) {
	d.mu.Lock()         // Lock the mutex
	defer d.mu.Unlock() // Unlock the mutex when the function returns

	// Close the currently opened database, if any. Best-effort like the
	// analogous close in Close() itself: the handle is discarded and replaced
	// right below regardless, but a failure here (e.g. a pooled connection
	// that would not release) is worth a log line rather than silence (B.13,
	// #181 — this used to swallow the error outright).
	if d.db != nil {
		if err := d.db.Close(); err != nil {
			slog.Warn("closing the previously open database failed", "err", err)
		}
	}

	// Take the single-writer lock before opening. If another instance holds it
	// this sets d.readOnly and we open read-only instead of racing a second
	// writer against the same file (the probable cause of the transient
	// "last database not reopened" failure — ADR-0004).
	d.acquireFileLock(path)

	// Neither the file lock nor an opened *sql.DB handle may leak on a
	// mid-open failure (pragmas, migration): any error below used to return
	// with the lock still held and (outside the read-only branch, which
	// already cleans up by hand) d.db still set, wedging the wrapper for any
	// later Setup/Open. Named-return style so every existing `return err`
	// below is covered without threading cleanup through each one; a no-op
	// when the read-only branch already closed d.db / never took the lock.
	defer func() {
		if err != nil {
			if d.db != nil {
				d.db.Close()
				d.db = nil
			}
			d.releaseFileLock()
		}
	}()

	// The PRAGMAs (foreign_keys=ON, busy_timeout, WAL, …) travel in the DSN
	// so the driver replays them on EVERY pooled connection. A PRAGMA run
	// after sql.Open configures only the one connection it lands on; with a
	// ten-connection pool the other nine ran with foreign_keys=OFF, so a
	// DeleteMatch served by one of them skipped ON DELETE CASCADE and left
	// game/move/move_analysis orphans (issue #157; blunderdb verify counts
	// them).
	d.db, err = sql.Open("sqlite", sqlite.DSN(path))
	if err != nil {
		return err
	}

	// Size the connection pool. Critical for ":memory:": each pooled
	// connection is a SEPARATE empty in-memory database, so concurrent reads
	// (RWMutex allows multiple readers) would otherwise hit a fresh connection
	// with no schema -> "no such table". ConfigurePool pins ":memory:" to a
	// single connection; file-backed DBs are allowed to grow.
	sqlite.ConfigurePool(d.db, path)

	// Read-only fallback: pin to a single connection so PRAGMA query_only (a
	// per-connection setting) reliably blocks writes on every query, then
	// forbid writes. The migration chain and ANALYZE both write, so they are
	// skipped — the writer instance that holds the lock owns them; it opened
	// with the same app version, so the schema is already current. The DSN
	// PRAGMAs (WAL included) run when the connection opens: the file is
	// writable here — only the single-writer lock is taken, by the other
	// instance, which has already put the file in WAL mode.
	if d.readOnly {
		d.db.SetMaxOpenConns(1)
		if _, err = d.db.Exec(`PRAGMA query_only = ON`); err != nil {
			return fmt.Errorf("cannot open database read-only: %w", err)
		}
		d.rebuildStore()
		return nil
	}

	migCtx, migDone := d.beginCancellableImport()
	defer migDone()
	if err := d.runMigrationChain(migCtx); err != nil {
		return err
	}

	d.ensureSearchStats()

	d.rebuildStore()
	return nil
}

// ensureSearchStats runs a one-time ANALYZE when the opened database has no
// query-planner statistics yet (sqlite_stat1 absent or empty). Without stats
// SQLite mis-estimates selectivity for non-selective search filters — e.g. a
// "win rate > 55% AND gammon > 20%" search that matches most rows is planned as
// a single-column analysis-index scan followed by a TEMP B-TREE sort on p.id,
// instead of scanning position in primary-key order (no sort). A full ANALYZE
// fixes the plan (~4x on that case in the tournois benchmark); the stats persist
// in the file, so later opens — and migrated databases, which already ANALYZE —
// skip this. Non-fatal: search still works with stale/absent stats.
func (d *Database) ensureSearchStats() {
	if d.db == nil {
		return
	}
	var n int
	// Errors (e.g. sqlite_stat1 does not exist yet) count as "no stats".
	if err := d.db.QueryRow(`SELECT count(*) FROM sqlite_stat1`).Scan(&n); err == nil && n > 0 {
		return
	}
	if _, err := d.db.Exec(`ANALYZE`); err != nil {
		slog.Warn("ANALYZE for search statistics failed", "err", err)
	}
}

// RefreshSearchStatistics runs a full ANALYZE, updating query-planner
// statistics for every table. Unlike ensureSearchStats (run automatically on
// open, but only when sqlite_stat1 is entirely empty), this always re-scans:
// after importing a batch of matches into an already-analysed database, the
// existing stats are stale rather than absent, so ensureSearchStats would
// skip them, silently leaving the planner working off pre-import row/value
// distributions. The CLI's batch importer has always run a plain `ANALYZE`
// after its file loop (cli_import.go, importBatch); this is the Wails-bound
// equivalent so the GUI's own batch import path
// (frontend/src/services/importService.js, importMultipleFiles) can do the
// same (fiche-05 T7). Best-effort and non-fatal, like ensureSearchStats: a
// search still works correctly, just possibly less optimally planned,
// without it.
//
// Takes the exclusive lock, like every other statement that writes to the
// database (ANALYZE rewrites sqlite_stat1/sqlite_stat4) — mirrors
// OpenDatabase, which holds d.mu.Lock() for the whole call including its own
// ensureSearchStats.
func (d *Database) RefreshSearchStatistics() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return
	}
	if _, err := d.db.Exec(`ANALYZE`); err != nil {
		slog.Warn("ANALYZE for search statistics failed", "err", err)
	}
}
