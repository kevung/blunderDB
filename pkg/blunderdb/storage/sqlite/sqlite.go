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
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open %q: %w", dsn, err)
	}
	if err := ApplyPragmas(db, dsn); err != nil {
		db.Close()
		return nil, err
	}
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

// ApplyPragmas applies the performance and safety PRAGMAs to a connection.
// WAL is skipped for in-memory databases (it needs a real filesystem). It is
// exported so the Database wrapper applies the exact same set (D9).
func ApplyPragmas(db *sql.DB, dsn string) error {
	if dsn != ":memory:" {
		var mode string
		if err := db.QueryRow(`PRAGMA journal_mode = WAL`).Scan(&mode); err != nil {
			return fmt.Errorf("sqlite: PRAGMA journal_mode=WAL: %w", err)
		}
	}
	for _, p := range []string{
		`PRAGMA synchronous  = NORMAL`,
		`PRAGMA cache_size   = -65536`,
		`PRAGMA temp_store   = MEMORY`,
		`PRAGMA mmap_size    = 268435456`,
		`PRAGMA foreign_keys = ON`,
	} {
		if _, err := db.Exec(p); err != nil {
			return fmt.Errorf("sqlite: %s: %w", p, err)
		}
	}
	return nil
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
// database is bootstrapped to the v2.7.0 schema; the 1.0→2.7 migration chain
// for pre-existing databases lands in P2 PR6.
func (s *Storage) Migrate(ctx context.Context) error {
	fresh, err := isFreshDB(ctx, s.sqlDB)
	if err != nil {
		return err
	}
	if fresh {
		return Bootstrap(ctx, s.sqlDB)
	}
	return nil
}
