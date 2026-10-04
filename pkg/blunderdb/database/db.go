package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/adrg/xdg"

	_ "modernc.org/sqlite"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcription"
)

type Database struct {
	db                *sql.DB
	mu                sync.RWMutex                        // RWMutex allows concurrent reads
	cancelMu          sync.Mutex                          // guards importCancel (held briefly, never with mu)
	importCancel      context.CancelFunc                  // cancels the in-flight import/migration; nil when idle
	searchCancels     map[int]context.CancelFunc          // in-flight searches, guarded by cancelMu
	searchSeq         int                                 // key of the next entry of searchCancels
	migrationProgress func(phase string, done, total int) // optional progress callback (GUI only)
	store             *sqlite.Storage                     // SQLite Storage backend, wraps db (P2)
	// importBatchID stamps every match the in-flight import writes, 0 when none
	// runs. One at a time, like importCancel.
	importBatchID int64
	// skipDuplicates makes an exact duplicate a plain skip again, without
	// offering its deeper analyses to the stored positions.
	skipDuplicates atomic.Bool
	// importBatchCounts accumulates what only the writing path sees; the caller
	// that opened the batch adds the unreadable files when it finishes it.
	importBatchCounts domain.ImportReport
	// importJournal is the journal of a resumed batch (ResumeImportBatch),
	// nil for a fresh one.
	importJournal *ingest.Journal
	// positionsSinceStats counts the positions imports have written since the
	// planner statistics were last refreshed (RefreshSearchStatistics).
	positionsSinceStats int
	// pendingPhaseBackfill is raised by the 2.19.0 migration step and cleared
	// by runMigrationChain once EnsureSchema has added position.game_phase.
	// A migration step cannot write a column the schema pass has not created
	// yet, and the phase backfill is the only 2.19.0 change that writes at all.
	pendingPhaseBackfill bool
	// pendingAnkiCardKinds is the same deferral for the 2.23.0 step, whose
	// repair also rebuilds anki_card: ALTER TABLE relaxes no constraint.
	pendingAnkiCardKinds bool
	lock                 *fileLock // single-writer advisory lock on the open file (nil for :memory:/read-only)
	readOnly             bool      // opened read-only because another instance holds the write lock
	commentAuthor        string    // who signs the comments written here (SetCommentAuthor); guarded by mu
	// transcriptSvc is the transcription service over this handle's store
	// (db_transcription.go), holding the open drafts' sessions; transcriptMu
	// guards the pointer.
	transcriptMu  sync.Mutex
	transcriptSvc *transcription.Service
	// directionMem is what the direction service keeps between calls: the clock forecasts and
	// the page catalogue. Per Database, so two open databases never share one language.
	directionMem service.Memory
	// generation counts the files this handle has opened, created or closed;
	// guarded by mu. A background job remembers it at its start and writes
	// nothing once it has moved: its positions belong to the previous file.
	generation uint64
	// beforeSwitch runs before Open, Setup and Close take mu, so the GUI can
	// stop and wait for the jobs writing the file about to go away (they need
	// mu to finish their last write). Guarded by switchMu.
	switchMu     sync.Mutex
	beforeSwitch func()
}

// ErrDatabaseChanged refuses a background job's write once another database
// has been opened (or this one closed) since the job started.
var ErrDatabaseChanged = errors.New("the open database changed while the job ran")

// SetBeforeSwitch registers fn, run before every OpenDatabase, SetupDatabase
// and Close; nil clears it. The GUI stops its batches there.
func (d *Database) SetBeforeSwitch(fn func()) {
	d.switchMu.Lock()
	defer d.switchMu.Unlock()
	d.beforeSwitch = fn
}

// switchFile runs the beforeSwitch hook; the caller then takes mu and calls
// bumpGeneration.
func (d *Database) switchFile() {
	d.switchMu.Lock()
	fn := d.beforeSwitch
	d.switchMu.Unlock()
	if fn != nil {
		fn()
	}
}

// bumpGeneration marks the file about to change; the caller holds mu.
func (d *Database) bumpGeneration() { d.generation++ }

// currentGeneration is the generation a background job remembers at its start.
func (d *Database) currentGeneration() uint64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.generation
}

// lockPathFor returns the file whose advisory lock guards a database against a
// second writer. It lives in the cache directory: a lock file beside the
// database reads as debris yet can never be unlinked on release (another
// instance may hold its inode, letting a third lock a fresh file). Named from
// the absolute path; falls back beside the database if the cache directory
// cannot be created (ADR-0004).
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

// acquireFileLock takes the single-writer advisory lock before a file-backed
// database is opened. If another instance holds it, d.readOnly is set and the
// caller must open read-only (ADR-0004: never read-write twice). A lock
// infrastructure failure is NON-fatal — the open proceeds unguarded.
// :memory: and the empty path are never locked.
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

// beginSearch gives a window, the count or a rank of the browsed search a
// context that CancelSearch can cancel from another goroutine, while it holds
// d.mu. A page of ids and its count run side by side, so every such call in
// flight is registered, not only the last.
// The returned done func must be deferred.
func (d *Database) beginSearch() (context.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	d.cancelMu.Lock()
	if d.searchCancels == nil {
		d.searchCancels = map[int]context.CancelFunc{}
	}
	key := d.searchSeq
	d.searchSeq++
	d.searchCancels[key] = cancel
	d.cancelMu.Unlock()
	return ctx, func() {
		d.cancelMu.Lock()
		delete(d.searchCancels, key)
		d.cancelMu.Unlock()
		cancel()
	}
}

// CancelSearch aborts the browsed search in flight — SearchPositionIDs,
// CountPositionsByFilters, IndexOfPositionByFilters — each returning an error
// wrapping context.Canceled. A whole id list (LoadPositionIDsByFilters) runs
// on: it serves a deck or a session, not the list Escape or a new search
// replaces. It is bound to the Wails frontend. No-op when idle.
func (d *Database) CancelSearch() {
	d.cancelMu.Lock()
	cancels := make([]context.CancelFunc, 0, len(d.searchCancels))
	for _, cancel := range d.searchCancels {
		cancels = append(cancels, cancel)
	}
	d.cancelMu.Unlock()
	for _, cancel := range cancels {
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

// conn returns the underlying *sql.DB handle, nil before Setup/Open.
// Unexported on purpose: *Database is bound wholesale to Wails, so an exported
// method would hand the raw handle to the frontend. Outside callers use RawConn.
func (d *Database) conn() *sql.DB {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db
}

// RawConn returns d's underlying *sql.DB handle for maintenance scripts and
// tests (e.g. scripts/demodb). A function, not a method, so the Wails binding
// never exposes it. Never call it from GUI-reachable code; add a named method.
func RawConn(d *Database) *sql.DB {
	return d.conn()
}

// Checkpoint truncates the WAL into the main file (wal_checkpoint(TRUNCATE)),
// keeping it bounded during a long batch import. Best-effort: a checkpoint
// blocked by a reader is no reason to fail the import, so the caller logs the
// error.
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
	d.switchFile()
	d.forgetTranscriptSessions()
	d.mu.Lock()
	defer d.mu.Unlock()
	d.bumpGeneration()
	d.releaseFileLock()
	wasReadOnly := d.readOnly
	d.readOnly = false
	if d.db == nil {
		return nil
	}
	// PRAGMA optimize before closing, as storage/sqlite's Close does (this
	// wrapper owns its own *sql.DB). Skipped read-only: query_only rejects the
	// write ANALYZE may attempt.
	if !wasReadOnly {
		_, _ = d.db.Exec(`PRAGMA optimize`)
	}
	err := d.db.Close()
	d.db = nil
	return err
}

func (d *Database) SetupDatabase(path string) (err error) {
	d.switchFile()
	d.forgetTranscriptSessions()
	d.mu.Lock()
	defer d.mu.Unlock()
	d.bumpGeneration()

	// Close the currently opened database, if any. Best-effort: the handle is
	// replaced below regardless, but a failure deserves a log line.
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

	// On any error below, roll back to a clean "never opened" state: a leaked
	// lock or handle would wedge every later Setup/Open. Named return, so each
	// `return err` is covered.
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
	// pool the others ran with foreign_keys=OFF and DeleteMatch skipped ON
	// DELETE CASCADE, leaving orphans.
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
	d.switchFile()
	d.forgetTranscriptSessions()
	d.mu.Lock()
	defer d.mu.Unlock()
	d.bumpGeneration()

	// Close the currently opened database, if any. Best-effort: the handle is
	// replaced below regardless, but a failure deserves a log line.
	if d.db != nil {
		if err := d.db.Close(); err != nil {
			slog.Warn("closing the previously open database failed", "err", err)
		}
	}

	// Take the single-writer lock first; if another instance holds it, open
	// read-only rather than race a second writer (ADR-0004).
	d.acquireFileLock(path)

	// On any error below, release the lock and handle so a later Setup/Open is
	// not wedged. Named return covers each `return err`; a no-op when the
	// read-only branch already cleaned up.
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
	// pool the others ran with foreign_keys=OFF and DeleteMatch skipped ON
	// DELETE CASCADE, leaving orphans.
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

	// Read-only fallback: one connection, so the per-connection query_only
	// blocks every write. Migration and ANALYZE are left to the writer instance
	// holding the lock, which already brought the schema current.
	if d.readOnly {
		d.db.SetMaxOpenConns(1)
		if _, err = d.db.Exec(`PRAGMA query_only = ON`); err != nil {
			return fmt.Errorf("cannot open database read-only: %w", err)
		}
		// The writer may be an older blunderDB that never migrated the file:
		// this instance cannot migrate it without writing (ADR-0007), and
		// reading an older schema with today's queries would fail table by
		// table, so it refuses up front.
		var onDisk string
		if err = d.db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&onDisk); err != nil {
			return fmt.Errorf("cannot open database read-only: reading its version: %w", err)
		}
		if cmp, cerr := compareVersions(onDisk, DatabaseVersion); cerr != nil || cmp != 0 {
			return fmt.Errorf("cannot open database read-only: its schema is %s and this blunderDB reads %s; "+
				"close the other instance holding it so this one can open it for writing", onDisk, DatabaseVersion)
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

// ensureSearchStats runs a one-time ANALYZE when sqlite_stat1 is absent or
// empty. Without stats SQLite plans a non-selective search filter as an index
// scan plus a TEMP B-TREE sort instead of a primary-key scan (~4x slower).
// The stats persist in the file. Non-fatal.
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

// statsRefreshMinPositions is how many imported positions it takes before
// RefreshSearchStatistics touches the planner statistics. A batch of a file
// or two barely moves the row-count ratios the planner reads; the count
// carries over from batch to batch, so small batches still add up to a
// refresh.
const statsRefreshMinPositions = 1000

// statsAnalysisLimit bounds how many index rows ANALYZE reads per index
// (PRAGMA analysis_limit): approximate statistics at a cost independent of
// the database size, instead of a scan of every index.
const statsAnalysisLimit = 1000

// RefreshSearchStatistics refreshes the planner statistics after an import
// batch — the stats are stale rather than absent, which ensureSearchStats
// would skip. It does nothing until statsRefreshMinPositions positions have
// been imported since the last refresh, and then runs PRAGMA optimize under
// PRAGMA analysis_limit: only the tables whose size has changed enough are
// analysed, each index sampled rather than scanned. 0x10002 asks optimize to
// consider every table, not only those this connection has queried. The GUI
// and CLI batch imports call it once per batch. Best-effort.
// Takes d.mu exclusively: ANALYZE writes sqlite_stat1/sqlite_stat4.
func (d *Database) RefreshSearchStatistics() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil || d.positionsSinceStats < statsRefreshMinPositions {
		return
	}
	ctx := context.Background()
	// analysis_limit is per connection: both pragmas run on the same one.
	conn, err := d.db.Conn(ctx)
	if err != nil {
		slog.Warn("planner statistics refresh: no connection", "err", err)
		return
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, fmt.Sprintf(`PRAGMA analysis_limit=%d`, statsAnalysisLimit)); err != nil {
		slog.Warn("PRAGMA analysis_limit failed", "err", err)
		return
	}
	if _, err := conn.ExecContext(ctx, `PRAGMA optimize=0x10002`); err != nil {
		slog.Warn("PRAGMA optimize for search statistics failed", "err", err)
		return
	}
	d.positionsSinceStats = 0
}
