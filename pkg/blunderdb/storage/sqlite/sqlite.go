// Package sqlite is the SQLite-backed implementation of storage.Storage. It
// uses the pure-Go modernc.org/sqlite driver (no CGO).
//
// Two constructors are exposed:
//
//   - Open creates/opens a database at a DSN and owns the *sql.DB: Close
//     closes it. This is the standalone library/CLI entry point.
//   - New wraps an existing *sql.DB handle without taking ownership: Close is
//     a no-op. The Database wrapper uses this so it keeps owning its handle
//     (in-memory GUI database, load/save to file).
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Storage is the SQLite implementation of storage.Storage.
type Storage struct {
	binder
	sqlDB  *sql.DB
	ownsDB bool
	// migrationProgress is Options.MigrationProgress, captured at Open time
	// (Migrate's signature is fixed by the storage.Storage interface, so it
	// cannot take one directly) and forwarded to the registered Migrator.
	migrationProgress func(phase string, done, total int)
}

var _ storage.Storage = (*Storage)(nil)

// Open opens (or creates) the SQLite database at dsn and returns a Storage
// that owns the connection. dsn ":memory:" yields an in-memory database.
func Open(ctx context.Context, dsn string, opts *storage.Options) (*Storage, error) {
	// Encode the PRAGMAs into the DSN so the driver applies them to *every*
	// connection the pool opens. A post-Open `PRAGMA` only configures the one
	// connection it runs on; the others would keep busy_timeout=0 and fail
	// concurrent writers with SQLITE_BUSY (P5).
	db, err := sql.Open("sqlite", DSN(dsn))
	if err != nil {
		return nil, fmt.Errorf("sqlite: open %q: %w", dsn, err)
	}
	ConfigurePool(db, dsn)
	fresh, err := isFreshDB(ctx, db)
	if err != nil {
		db.Close()
		return nil, err
	}
	if fresh {
		if err := Bootstrap(ctx, db); err != nil {
			db.Close()
			return nil, err
		}
	}
	var progress func(phase string, done, total int)
	if opts != nil {
		progress = opts.MigrationProgress
	}
	return &Storage{binder: binder{db: db}, sqlDB: db, ownsDB: true, migrationProgress: progress}, nil
}

// New wraps an existing *sql.DB handle. The returned Storage does not own the
// connection: Close is a no-op and the caller stays responsible for closing
// db. Used by the Database wrapper.
func New(db *sql.DB) *Storage {
	return &Storage{binder: binder{db: db}, sqlDB: db, ownsDB: false}
}

// perConnPragmas are the connection-scoped PRAGMAs every SQLite connection
// must carry. A PRAGMA only affects the connection it runs on, so they are
// encoded into the DSN (see DSN) and replayed by the driver on every
// connection the pool opens; there is deliberately no "apply to an open
// handle" helper any more — the one that existed configured a single pooled
// connection and left the others with foreign_keys=OFF (issue #157).
// busy_timeout makes a contending writer wait up to 10 s for the write lock
// rather than failing immediately with SQLITE_BUSY — essential now that the
// global Database mutex no longer serializes writers (P5). 10 s (not gnubg's
// usual 5 s) because Windows file locking is measurably slower under the
// heavy concurrent-writer load the storagetest suite exercises; positionStore
// Save additionally retries a handful of times with a short backoff, which
// busy_timeout alone cannot cover — see its doc comment.
var perConnPragmas = [][2]string{
	{"busy_timeout", "10000"},
	{"foreign_keys", "ON"},
	{"synchronous", "NORMAL"},
	{"cache_size", "-65536"},
	{"temp_store", "MEMORY"},
	{"mmap_size", "268435456"},
}

// DSN augments a SQLite path/DSN with the per-connection PRAGMAs (and, for a
// file-backed database, WAL journal mode) encoded as `_pragma` query params.
// The modernc driver runs these on every connection it opens, so the whole
// pool is configured identically — unlike a one-shot post-Open PRAGMA, which
// only configures a single connection. WAL is omitted for ":memory:" (it
// needs a real filesystem). Every sql.Open("sqlite", …) in blunderDB goes
// through here: the desktop/CLI Database wrapper as well as Open above.
//
// The driver splits its DSN at the first '?' and hands the left part to
// SQLite verbatim — no percent-decoding — so a plain path (spaces, accents,
// Windows drive letters and backslashes) is passed through untouched. Three
// shapes are told apart:
//
//   - ":memory:" or a path without '?': the path is used as is.
//   - a "file:" URI supplied by the caller (it may already carry a query such
//     as mode=ro): the PRAGMAs are appended to its query.
//   - a bare path containing '?': the driver would truncate it at the '?'
//     and open a different file, so it is rewritten as a percent-encoded
//     "file:" URI (SQLITE_OPEN_URI is set by the driver). Windows forbids
//     '?' in file names, so this only ever happens on Unix.
func DSN(path string) string {
	q := url.Values{}
	for _, p := range perConnPragmas {
		q.Add("_pragma", p[0]+"("+p[1]+")")
	}
	if path != ":memory:" {
		q.Add("_pragma", "journal_mode(WAL)")
	}
	base, sep := path, "?"
	switch {
	case strings.HasPrefix(path, "file:"):
		if strings.Contains(path, "?") {
			sep = "&"
		}
	case strings.Contains(path, "?"):
		base = fileURI(path)
	}
	return base + sep + q.Encode()
}

// fileURI turns a bare filesystem path into a SQLite "file:" URI, escaping
// every path segment so that '?', '#', '%' and non-ASCII bytes survive the
// driver's DSN split and SQLite's own URI decoding. An absolute path becomes
// "file:///…" (a Windows drive letter is given the leading slash SQLite
// expects); a relative one "file:rel/ative.db", which SQLite resolves against
// the working directory exactly like the bare path would be.
func fileURI(path string) string {
	segs := strings.Split(filepath.ToSlash(path), "/")
	for i, s := range segs {
		segs[i] = url.PathEscape(s)
	}
	enc := strings.Join(segs, "/")
	if filepath.IsAbs(path) {
		if !strings.HasPrefix(enc, "/") {
			enc = "/" + enc
		}
		return "file://" + enc
	}
	return "file:" + enc
}

// ConfigurePool sizes the *sql.DB connection pool for the given DSN.
//
// An ":memory:" database lives inside a single connection — each pooled
// connection would be a separate, empty database — so the pool is pinned to a
// single connection. A file-backed database is shared across connections, so
// the pool is allowed to grow (SQLite still serializes writers; busy_timeout
// makes contending writers wait rather than error).
func ConfigurePool(db *sql.DB, dsn string) {
	if dsn == ":memory:" {
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		return
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)
}

// Close closes the connection when this Storage owns it (Open); it is a no-op
// for a borrowed handle (New).
func (s *Storage) Close() error {
	if !s.ownsDB || s.sqlDB == nil {
		return nil
	}
	// PRAGMA optimize (SQLite docs: run before closing every long-lived
	// connection) is a cheap, targeted ANALYZE: it only touches tables whose
	// content has changed enough since the last full ANALYZE to plausibly have
	// stale sqlite_stat1 rows, so it is a no-op on a connection that did
	// little writing and inexpensive even after a heavy import (fiche-05 T7).
	// Best-effort: a failed optimize must not turn a normal shutdown into an
	// error the caller has no useful way to react to.
	_, _ = s.sqlDB.Exec(`PRAGMA optimize`)
	return s.sqlDB.Close()
}

// BeginTx starts a transaction whose family accessors run inside it.
func (s *Storage) BeginTx(ctx context.Context) (storage.Tx, error) {
	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("sqlite: begin tx: %w", err)
	}
	return &txImpl{binder: binder{db: tx}, tx: tx}, nil
}

// Version reports the schema version recorded in the metadata table. It
// delegates to MetadataStore.Version (D6).
func (s *Storage) Version(ctx context.Context) (string, error) {
	return s.Metadata().Version(ctx, "")
}

// Migrate brings the database up to the current schema version. A fresh
// database is bootstrapped to the current schema; a pre-existing database is
// upgraded in place through the registered legacy migration chain (P2 PR6).
// When no migrator is registered, a non-fresh database can only be opened if
// it is already current: Migrate compares its recorded version against
// domain.DatabaseVersion and errors out rather than silently leaving an older
// database un-migrated (a pure-library consumer that never imports package
// database, such as cmd/serve without the blank import, must not pretend to
// have upgraded a database it cannot actually migrate).
func (s *Storage) Migrate(ctx context.Context) error {
	fresh, err := isFreshDB(ctx, s.sqlDB)
	if err != nil {
		return err
	}
	if fresh {
		return Bootstrap(ctx, s.sqlDB)
	}
	if registeredMigrator != nil {
		return registeredMigrator(ctx, s.sqlDB, s.migrationProgress)
	}
	version, err := s.Metadata().Version(ctx, "")
	if err != nil {
		return fmt.Errorf("sqlite: read schema version of non-fresh database: %w", err)
	}
	if version != domain.DatabaseVersion {
		return fmt.Errorf("sqlite: database is at schema version %q, need %q, and no migrator is registered (blank-import package database, e.g. `_ \"github.com/kevung/blunderdb/pkg/blunderdb/database\"`, to enable migrating older databases)", version, domain.DatabaseVersion)
	}
	return nil
}
