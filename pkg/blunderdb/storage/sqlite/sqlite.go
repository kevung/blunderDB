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
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Storage is the SQLite implementation of storage.Storage.
type Storage struct {
	binder
	sqlDB  *sql.DB
	ownsDB bool
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
	return &Storage{binder: binder{db: db}, sqlDB: db, ownsDB: true}, nil
}

// New wraps an existing *sql.DB handle. The returned Storage does not own the
// connection: Close is a no-op and the caller stays responsible for closing
// db. Used by the Database wrapper.
func New(db *sql.DB) *Storage {
	return &Storage{binder: binder{db: db}, sqlDB: db, ownsDB: false}
}

// perConnPragmas are the connection-scoped PRAGMAs every SQLite connection
// must carry. They are set per connection (via the DSN, or via ApplyPragmas on
// a borrowed handle) because a PRAGMA only affects the connection it runs on.
// busy_timeout makes a contending writer wait up to 5 s for the write lock
// rather than failing immediately with SQLITE_BUSY — essential now that the
// global Database mutex no longer serializes writers (P5).
var perConnPragmas = [][2]string{
	{"busy_timeout", "5000"},
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
// needs a real filesystem).
func DSN(path string) string {
	q := url.Values{}
	for _, p := range perConnPragmas {
		q.Add("_pragma", p[0]+"("+p[1]+")")
	}
	if path != ":memory:" {
		q.Add("_pragma", "journal_mode(WAL)")
	}
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	return path + sep + q.Encode()
}

// ApplyPragmas applies the performance and safety PRAGMAs to an already-open
// handle. WAL is skipped for in-memory databases (it needs a real filesystem).
// It is exported so the Database wrapper applies the exact same set (D9) to the
// handle it owns. Prefer opening via a DSN built with DSN() when possible, so
// every pooled connection is configured (this helper only touches whichever
// connection the driver hands it).
func ApplyPragmas(db *sql.DB, dsn string) error {
	if dsn != ":memory:" {
		var mode string
		if err := db.QueryRow(`PRAGMA journal_mode = WAL`).Scan(&mode); err != nil {
			return fmt.Errorf("sqlite: PRAGMA journal_mode=WAL: %w", err)
		}
	}
	for _, p := range perConnPragmas {
		stmt := fmt.Sprintf(`PRAGMA %s = %s`, p[0], p[1])
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("sqlite: %s: %w", stmt, err)
		}
	}
	return nil
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
// When no migrator is registered (pure-library consumer importing only this
// package) a non-fresh database is left untouched — assumed already current.
func (s *Storage) Migrate(ctx context.Context) error {
	fresh, err := isFreshDB(ctx, s.sqlDB)
	if err != nil {
		return err
	}
	if fresh {
		return Bootstrap(ctx, s.sqlDB)
	}
	if registeredMigrator != nil {
		return registeredMigrator(ctx, s.sqlDB)
	}
	return nil
}
