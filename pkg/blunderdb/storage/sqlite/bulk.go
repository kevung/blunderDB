package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// A bulk session writes a large import through one dedicated connection
// tuned for throughput. Every position written updates some thirty B-trees;
// once parsing is parallel, the import is bound by index I/O, and a 64 MB page
// cache cannot hold indexes that grow to several gigabytes.
//
// The PRAGMAs a bulk session changes are connection-scoped, which is why the
// session owns its connection: set on a pooled one, they would reach whichever
// connection the next transaction happened to draw.

// bulkPragmas are applied to the session's connection; perConnPragmas (or
// SQLite's default for wal_autocheckpoint) are restored on Close.
var bulkPragmas = [][2]string{
	{"cache_size", "-524288"},       // 512 MiB
	{"wal_autocheckpoint", "50000"}, // pages: checkpoint every ~200 MB of WAL, not every 4 MB
}

// unsafeBulkPragmas trade durability for speed: a power cut during the import
// can corrupt the database. Only for a database that holds nothing yet.
var unsafeBulkPragmas = [][2]string{
	{"synchronous", "OFF"},
}

// sqliteDefaultWALAutocheckpoint is SQLite's own default, restored on Close.
const sqliteDefaultWALAutocheckpoint = "1000"

// BulkOptions chooses what a bulk session may give up.
type BulkOptions struct {
	// Unsafe sets synchronous=OFF: a cut during the import means rebuilding
	// the database. Only for a database the import is filling from empty.
	Unsafe bool
	// DropIndexes drops the secondary indexes of position and analysis before
	// the import and rebuilds them on Close; the Zobrist index and the
	// analysis-per-position index stay, since deduplication reads them. A cut
	// leaves them missing until the next open, where EnsureSchema rebuilds them.
	DropIndexes bool
}

// BulkSession is a storage.Tx source bound to one tuned connection.
type BulkSession struct {
	db      *sql.DB
	conn    *sql.Conn
	opts    BulkOptions
	dropped []string // CREATE statements of the indexes dropped
}

// bulkKeptIndexes are read by the import itself and never dropped.
var bulkKeptIndexes = map[string]bool{
	"idx_position_zobrist":  true,
	"idx_analysis_position": true,
}

var indexStmt = regexp.MustCompile(`^CREATE\s+(?:UNIQUE\s+)?INDEX IF NOT EXISTS (\w+)\s+ON (\w+)\(`)

// bulkDroppableIndexes returns name → CREATE statement of the secondary
// indexes of position and analysis, from the one schema.
func bulkDroppableIndexes() map[string]string {
	out := map[string]string{}
	for _, stmt := range schemaStatements {
		m := indexStmt.FindStringSubmatch(stmt)
		if m == nil || bulkKeptIndexes[m[1]] || (m[2] != "position" && m[2] != "analysis") {
			continue
		}
		out[m[1]] = stmt
	}
	return out
}

// BeginBulk opens a bulk session on db. The caller must Close it, even after
// an error in between: Close restores the connection and rebuilds the indexes.
func BeginBulk(ctx context.Context, db *sql.DB, opts BulkOptions) (*BulkSession, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("sqlite: bulk connection: %w", err)
	}
	s := &BulkSession{db: db, conn: conn, opts: opts}
	pragmas := bulkPragmas
	if opts.Unsafe {
		pragmas = append(append([][2]string{}, pragmas...), unsafeBulkPragmas...)
	}
	for _, p := range pragmas {
		if _, err := conn.ExecContext(ctx, "PRAGMA "+p[0]+" = "+p[1]); err != nil {
			_ = s.Close(context.Background())
			return nil, fmt.Errorf("sqlite: bulk pragma %s: %w", p[0], err)
		}
	}
	if opts.DropIndexes {
		for name, stmt := range bulkDroppableIndexes() {
			if _, err := conn.ExecContext(ctx, "DROP INDEX IF EXISTS "+name); err != nil {
				_ = s.Close(context.Background())
				return nil, fmt.Errorf("sqlite: bulk drop index %s: %w", name, err)
			}
			s.dropped = append(s.dropped, stmt)
		}
	}
	slog.Info("sqlite: bulk import session", "unsafe", opts.Unsafe, "droppedIndexes", len(s.dropped))
	return s, nil
}

// BeginTx starts a transaction on the session's connection.
func (s *BulkSession) BeginTx(ctx context.Context) (storage.Tx, error) {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("sqlite: begin tx: %w", err)
	}
	return newTxImpl(tx), nil
}

// Close rebuilds the dropped indexes, restores the connection's PRAGMAs to
// the pool's and returns it. It uses its own context: a cancelled import must
// still leave its indexes and PRAGMAs as they were.
func (s *BulkSession) Close(ctx context.Context) error {
	if s.conn == nil {
		return nil
	}
	var firstErr error
	keep := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	for _, stmt := range s.dropped {
		if _, err := s.conn.ExecContext(ctx, stmt); err != nil {
			keep(fmt.Errorf("sqlite: rebuild index: %w", err))
		}
	}
	restore := map[string]string{"wal_autocheckpoint": sqliteDefaultWALAutocheckpoint}
	for _, p := range perConnPragmas {
		restore[p[0]] = p[1]
	}
	for _, p := range append(append([][2]string{}, bulkPragmas...), unsafeBulkPragmas...) {
		if _, err := s.conn.ExecContext(ctx, "PRAGMA "+p[0]+" = "+restore[p[0]]); err != nil {
			keep(fmt.Errorf("sqlite: restore pragma %s: %w", p[0], err))
		}
	}
	_, err := s.conn.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)")
	keep(err)
	keep(s.conn.Close())
	s.conn = nil
	return firstErr
}
