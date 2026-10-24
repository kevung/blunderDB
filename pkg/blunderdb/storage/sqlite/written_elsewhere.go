package sqlite

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	sqlitedriver "modernc.org/sqlite"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

var _ storage.WriteWatcher = (*Storage)(nil)

// OpenDB opens the database at path (shaped by DSN) as a pool that can tell
// another process's writes from its own: see WrittenElsewhere. Opening
// connects nothing yet, exactly like sql.Open. An in-memory database has no
// other writer and is opened plainly.
func OpenDB(path string) (*sql.DB, error) {
	dsn := DSN(path)
	if path == ":memory:" {
		return sql.Open("sqlite", dsn)
	}
	base, err := sqlitedriver.NewConnector(dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open %q: %w", path, err)
	}
	return sql.OpenDB(&watchedConnector{base: base, watch: &writeWatch{base: base}}), nil
}

// CloseDB closes db and, for a pool opened by OpenDB, the connection that
// watches it: database/sql never closes a connection it does not pool.
func CloseDB(db *sql.DB) error {
	if w := watchOf(db); w != nil {
		w.close()
	}
	return db.Close()
}

// WrittenElsewhere reports whether another process (the CLI, a serve daemon)
// committed to the file since the previous call; the first call only takes the
// mark. The pool's own commits are not reported: whoever made them already
// knows. When both happen between two calls, the change is taken as the
// pool's own — PRAGMA data_version says that the file changed, not how many
// times — so such an outside write is reported only by the next one. A pool
// not opened by OpenDB reports nothing. Read-only: data_version is read on a
// connection of its own, which never writes (ADR-0007).
func (s *Storage) WrittenElsewhere(ctx context.Context) (bool, error) {
	w := watchOf(s.sqlDB)
	if w == nil {
		return false, nil
	}
	return w.changed(ctx)
}

func watchOf(db *sql.DB) *writeWatch {
	if db == nil {
		return nil
	}
	w, _ := db.Driver().(*writeWatch)
	return w
}

// watchedConnector counts every commit of the pool's connections through
// SQLite's commit hook, which runs before the commit is visible to any other
// connection.
type watchedConnector struct {
	base  driver.Connector
	watch *writeWatch
}

func (c *watchedConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.base.Connect(ctx)
	if err != nil {
		return nil, err
	}
	if hr, ok := conn.(sqlitedriver.HookRegisterer); ok {
		w := c.watch
		hr.RegisterCommitHook(func() int32 {
			w.own.Add(1)
			return 0
		})
	}
	return conn, nil
}

// Driver is how a *sql.DB leads back to its watch (sql.DB.Driver).
func (c *watchedConnector) Driver() driver.Driver { return c.watch }

// writeWatch holds a connection outside the pool, which only ever reads
// PRAGMA data_version. That value is per connection and moves at every commit
// of any other connection — at the pool's own commits too, which the commit
// hook counts in own: a move with no own commit since the last one came from
// elsewhere.
type writeWatch struct {
	base driver.Connector
	own  atomic.Uint64

	mu      sync.Mutex
	conn    driver.Conn
	version int64
	seenOwn uint64
	closed  bool
}

// Open makes writeWatch the driver.Driver that sql.DB.Driver returns; a
// connection opened this way is not counted, and database/sql never asks.
func (w *writeWatch) Open(name string) (driver.Conn, error) {
	return w.base.Driver().Open(name)
}

func (w *writeWatch) changed(ctx context.Context) (bool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return false, nil
	}
	if w.conn == nil {
		conn, err := w.base.Connect(ctx)
		if err != nil {
			return false, fmt.Errorf("sqlite: watch writes: %w", err)
		}
		v, err := dataVersion(ctx, conn)
		if err != nil {
			_ = conn.Close()
			return false, err
		}
		w.conn, w.version, w.seenOwn = conn, v, w.own.Load()
		return false, nil
	}
	v, err := dataVersion(ctx, w.conn)
	if err != nil {
		// The next call starts again from a fresh mark.
		_ = w.conn.Close()
		w.conn = nil
		return false, err
	}
	if v == w.version {
		return false, nil
	}
	// own is read after data_version: a commit that moved it had its hook run
	// before, so it is counted. A commit counted but not visible yet is taken
	// for an outside one at the next move: one read too many, never one missed.
	own := w.own.Load()
	elsewhere := own == w.seenOwn
	w.version, w.seenOwn = v, own
	return elsewhere, nil
}

func (w *writeWatch) close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	if w.conn != nil {
		_ = w.conn.Close()
		w.conn = nil
	}
}

func dataVersion(ctx context.Context, conn driver.Conn) (int64, error) {
	q, ok := conn.(driver.QueryerContext)
	if !ok {
		return 0, errors.New("sqlite: watch writes: the driver cannot query")
	}
	rows, err := q.QueryContext(ctx, `PRAGMA data_version`, nil)
	if err != nil {
		return 0, fmt.Errorf("sqlite: read data_version: %w", err)
	}
	defer rows.Close()
	dest := make([]driver.Value, 1)
	if err := rows.Next(dest); err != nil {
		return 0, fmt.Errorf("sqlite: read data_version: %w", err)
	}
	v, ok := dest[0].(int64)
	if !ok {
		return 0, fmt.Errorf("sqlite: read data_version: unexpected %T", dest[0])
	}
	return v, nil
}
