package sqlite

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"sync"

	sqlitedriver "modernc.org/sqlite"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

var _ storage.WriteWatcher = (*Storage)(nil)

// How another process's commits are told from the pool's own.
//
// PRAGMA data_version is per connection: it moves when any other connection
// committed since that connection last looked, never for its own commits.
// One connection outside the pool, the watch, only ever reads it; it moves at
// the pool's commits as well as at another process's. Every commit of the
// pool is therefore accounted for, so that the watch's mark (its value when
// every change up to it was accounted for) only ever moves over a change
// known to be the pool's own:
//
//   - In the commit hook the committing connection holds the write lock: no
//     one else can commit until it lets go. The watch is read there; a move
//     since the mark is a commit not accounted for, and is reported.
//   - Once the connection is handed back to the pool, the watch is read
//     again, then the committing connection's own data_version. If that one
//     has not moved since the connection left the pool, no other connection
//     committed meanwhile, so the watch's move is exactly the pool's commit:
//     the mark follows it. Otherwise the mark stays and the next call reports
//     a change.
//
// Nothing is ever missed; what cannot be told apart is reported: commits of
// two connections of the pool at once, a checkpoint or an ANALYZE of another
// process, a VACUUM. One read too many, never one write missed.

// OpenDB opens the database at path (shaped by DSN) as a pool that can tell
// another process's writes from its own: see WrittenElsewhere. The watch
// takes its mark at once, so that a write made before the first call is not
// lost; failing that, the first call reports a change. An in-memory database
// has no other writer and is opened plainly.
func OpenDB(path string) (*sql.DB, error) {
	if path == ":memory:" {
		return sql.Open("sqlite", DSN(path))
	}
	base, err := sqlitedriver.NewConnector(DSN(path))
	if err != nil {
		return nil, fmt.Errorf("sqlite: open %q: %w", path, err)
	}
	watchBase, err := sqlitedriver.NewConnector(dsn(path, false))
	if err != nil {
		return nil, fmt.Errorf("sqlite: open %q: %w", path, err)
	}
	w := &writeWatch{base: base, watchBase: watchBase}
	w.mu.Lock()
	w.markLocked(context.Background())
	w.mu.Unlock()
	return sql.OpenDB(w), nil
}

// WrittenElsewhere reports whether another process (the CLI, a serve daemon)
// committed to the file since the previous call, or since the pool was
// opened. The pool's own commits are not reported, except when they cannot be
// told apart (see above). A pool not opened by OpenDB reports nothing.
// Read-only: data_version is read on a connection that never writes
// (ADR-0007).
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

// writeWatch is the pool's connector (and its driver, how a *sql.DB leads
// back to it) and the watch connection's keeper.
type writeWatch struct {
	base      driver.Connector // the pool's connections
	watchBase driver.Connector // the watch: no WAL switch, it never writes

	mu       sync.Mutex
	conn     driver.Conn // the watch, nil until connected or after an error
	stmt     driver.Stmt // PRAGMA data_version on conn
	mark     int64
	known    bool // mark accounts for every commit up to it
	pending  bool // a commit not accounted for was seen and not reported yet
	inFlight int  // connections that committed and are not back in the pool
	closed   bool
}

// Connect opens a pool connection that tells the watch about its commits.
func (w *writeWatch) Connect(ctx context.Context) (driver.Conn, error) {
	inner, err := w.base.Connect(ctx)
	if err != nil {
		return nil, err
	}
	c := &watchedConn{inner: inner, watch: w}
	if c.stmt, err = prepareDataVersion(ctx, inner); err != nil {
		_ = inner.Close()
		return nil, err
	}
	if hr, ok := inner.(sqlitedriver.HookRegisterer); ok {
		hr.RegisterCommitHook(c.committing)
	}
	c.look(ctx)
	return c, nil
}

// Driver makes the watch what sql.DB.Driver returns.
func (w *writeWatch) Driver() driver.Driver { return w }

// Open is never called by database/sql on a connector's pool; a connection
// opened this way is not watched.
func (w *writeWatch) Open(name string) (driver.Conn, error) {
	return w.base.Driver().Open(name)
}

// readLocked reads the watch's data_version, connecting it if needed. On an
// error the watch is dropped and the mark is no longer known.
func (w *writeWatch) readLocked(ctx context.Context) (int64, error) {
	if w.conn == nil {
		conn, err := w.watchBase.Connect(ctx)
		if err != nil {
			w.known = false
			return 0, fmt.Errorf("sqlite: watch writes: %w", err)
		}
		stmt, err := prepareDataVersion(ctx, conn)
		if err != nil {
			_ = conn.Close()
			w.known = false
			return 0, err
		}
		w.conn, w.stmt = conn, stmt
	}
	v, err := readDataVersion(ctx, w.stmt)
	if err != nil {
		w.dropLocked()
		w.known = false
		return 0, err
	}
	return v, nil
}

func (w *writeWatch) dropLocked() {
	if w.stmt != nil {
		_ = w.stmt.Close()
		w.stmt = nil
	}
	if w.conn != nil {
		_ = w.conn.Close()
		w.conn = nil
	}
}

func (w *writeWatch) markLocked(ctx context.Context) {
	if v, err := w.readLocked(ctx); err == nil {
		w.mark, w.known = v, true
	}
}

// committed is the commit hook's side: the committing connection holds the
// write lock, so the watch shows every commit before this one and no other.
// first says the connection had not committed since it left the pool.
func (w *writeWatch) committed(first bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return
	}
	if first {
		w.inFlight++
	}
	v, err := w.readLocked(context.Background())
	if err != nil {
		return
	}
	if !w.known || v != w.mark {
		w.pending = true
	}
	w.mark, w.known = v, true
}

// settled is a connection that committed coming back to the pool: clean says
// its own data_version did not move since it left the pool, read after the
// watch's (the read callback), so the watch's move is its commit alone.
func (w *writeWatch) settled(read func() (clean bool)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.inFlight > 0 {
		w.inFlight--
	}
	if w.closed || !w.known {
		return
	}
	v, err := w.readLocked(context.Background())
	if err != nil {
		return
	}
	if read != nil && read() {
		w.mark = v
	}
}

func (w *writeWatch) changed(ctx context.Context) (bool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return false, nil
	}
	if w.inFlight > 0 {
		// A commit of the pool is not settled yet: its end will move the
		// mark, or leave it for the next call to report.
		report := w.pending
		w.pending = false
		return report, nil
	}
	v, err := w.readLocked(ctx)
	if err != nil {
		return false, err
	}
	report := w.pending || !w.known || v != w.mark
	w.pending, w.mark, w.known = false, v, true
	return report, nil
}

// Close closes the watch connection: sql.DB.Close calls it on its connector
// (io.Closer), since database/sql never closes a connection it does not pool.
func (w *writeWatch) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	w.dropLocked()
	return nil
}

// watchedConn is a pool connection; the methods database/sql uses are passed
// through, and the pool's hand-back (IsValid) settles a commit.
type watchedConn struct {
	inner driver.Conn
	watch *writeWatch
	stmt  driver.Stmt // PRAGMA data_version on inner

	// own is the connection's data_version when it left the pool; ownKnown
	// is false when it could not be read. hooked says it committed since.
	// All three are touched only by the goroutine database/sql gives the
	// connection to, the commit hook included.
	own      int64
	ownKnown bool
	hooked   bool
}

var (
	_ driver.ConnBeginTx        = (*watchedConn)(nil)
	_ driver.ConnPrepareContext = (*watchedConn)(nil)
	_ driver.ExecerContext      = (*watchedConn)(nil)
	_ driver.QueryerContext     = (*watchedConn)(nil)
	_ driver.Pinger             = (*watchedConn)(nil)
	_ driver.SessionResetter    = (*watchedConn)(nil)
	_ driver.Validator          = (*watchedConn)(nil)
)

func (c *watchedConn) committing() int32 {
	first := !c.hooked
	c.hooked = true
	c.watch.committed(first)
	return 0
}

// look reads the connection's own data_version.
func (c *watchedConn) look(ctx context.Context) {
	v, err := readDataVersion(ctx, c.stmt)
	c.own, c.ownKnown = v, err == nil
}

// settle runs where the connection is idle again after a commit.
func (c *watchedConn) settle(usable bool) {
	if !c.hooked {
		return
	}
	c.hooked = false
	if !usable {
		c.watch.settled(nil)
		return
	}
	before, knew := c.own, c.ownKnown
	c.watch.settled(func() bool {
		c.look(context.Background())
		return knew && c.ownKnown && c.own == before
	})
}

func (c *watchedConn) Prepare(query string) (driver.Stmt, error) { return c.inner.Prepare(query) }

func (c *watchedConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if p, ok := c.inner.(driver.ConnPrepareContext); ok {
		return p.PrepareContext(ctx, query)
	}
	return c.inner.Prepare(query)
}

func (c *watchedConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *watchedConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if b, ok := c.inner.(driver.ConnBeginTx); ok {
		return b.BeginTx(ctx, opts)
	}
	return c.inner.Begin() //nolint:staticcheck // the fallback database/sql itself would take
}

func (c *watchedConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if e, ok := c.inner.(driver.ExecerContext); ok {
		return e.ExecContext(ctx, query, args)
	}
	return nil, driver.ErrSkip
}

func (c *watchedConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if q, ok := c.inner.(driver.QueryerContext); ok {
		return q.QueryContext(ctx, query, args)
	}
	return nil, driver.ErrSkip
}

func (c *watchedConn) Ping(ctx context.Context) error {
	if p, ok := c.inner.(driver.Pinger); ok {
		return p.Ping(ctx)
	}
	return nil
}

// ResetSession runs when the pool hands the connection out again: its own
// data_version is taken then, the reference settle compares against.
func (c *watchedConn) ResetSession(ctx context.Context) error {
	if r, ok := c.inner.(driver.SessionResetter); ok {
		if err := r.ResetSession(ctx); err != nil {
			return err
		}
	}
	c.look(ctx)
	return nil
}

// IsValid runs when the connection comes back to the pool, idle.
func (c *watchedConn) IsValid() bool {
	valid := true
	if v, ok := c.inner.(driver.Validator); ok {
		valid = v.IsValid()
	}
	c.settle(valid)
	return valid
}

func (c *watchedConn) Close() error {
	c.settle(false)
	// The driver keeps the hook in a table of its own that Close does not
	// empty.
	if hr, ok := c.inner.(sqlitedriver.HookRegisterer); ok {
		hr.RegisterCommitHook(nil)
	}
	if c.stmt != nil {
		_ = c.stmt.Close()
	}
	return c.inner.Close()
}

func prepareDataVersion(ctx context.Context, conn driver.Conn) (driver.Stmt, error) {
	p, ok := conn.(driver.ConnPrepareContext)
	if !ok {
		return nil, errors.New("sqlite: watch writes: the driver cannot prepare")
	}
	stmt, err := p.PrepareContext(ctx, `PRAGMA data_version`)
	if err != nil {
		return nil, fmt.Errorf("sqlite: prepare data_version: %w", err)
	}
	return stmt, nil
}

func readDataVersion(ctx context.Context, stmt driver.Stmt) (int64, error) {
	q, ok := stmt.(driver.StmtQueryContext)
	if !ok {
		return 0, errors.New("sqlite: watch writes: the driver cannot query")
	}
	rows, err := q.QueryContext(ctx, nil)
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
