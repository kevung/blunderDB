package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"sync/atomic"
	"time"

	sqlitedriver "modernc.org/sqlite"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// shared adapts the backend's execer (a *sql.DB or a *sql.Tx) to the
// sqlshared.Execer the backend-agnostic stores run against, and answers the
// dialect questions for SQLite: '?' placeholders as-is, no tenant column on the
// domain tables, text dates and integer aggregates that need no cast.
type shared struct {
	db    execer
	probe *writeProbe
}

var _ sqlshared.Execer = shared{}

// shared returns the binder's execer wrapped for the shared stores.
func (b binder) shared() sqlshared.Execer { return shared(b) }

func (shared) Name() string                             { return "sqlite" }
func (shared) ScopeColumn() string                      { return "scope" }
func (shared) LibrarySettingsTable() (string, bool)     { return "metadata", false }
func (shared) ScopeArg(scope string) any                { return scope }
func (shared) TenantFilter(_, _ string) (string, []any) { return "1=1", nil }
func (shared) TenantColumns(string) ([]string, []any)   { return nil, nil }
func (shared) Bool(col string, v bool) string {
	if v {
		return col + " = 1"
	}
	return col + " = 0"
}
func (shared) BoolAsInt(col string) string { return col }
func (shared) BoolArg(v bool) any {
	if v {
		return 1
	}
	return 0
}
func (shared) Bigint(expr string) string { return expr }
func (shared) ILike() string             { return "LIKE" }
func (shared) ForUpdate() string         { return "" }
func (shared) Guard(string) (string, []any) {
	return "", nil
}
func (shared) LimitOffset(limit, offset int) (string, []any) {
	switch {
	case limit <= 0 && offset <= 0:
		return "", nil
	case limit <= 0:
		// SQLite's grammar requires a LIMIT whenever OFFSET is given; -1
		// means unbounded (SQLite's own documented convention).
		return " LIMIT -1 OFFSET ?", []any{offset}
	case offset <= 0:
		return " LIMIT ?", []any{limit}
	default:
		return " LIMIT ? OFFSET ?", []any{limit, offset}
	}
}
func (shared) TimestampArg() string                 { return "?" }
func (shared) TimestampTextArg() string             { return "?" }
func (shared) InstantArg(t time.Time) (string, any) { return "?", t.Unix() }
func (shared) DateText(col string) string           { return "COALESCE(SUBSTR(" + col + ",1,10),'')" }
func (shared) TimestampText(col string) string      { return "COALESCE(" + col + ",'')" }
func (shared) Referenced(err error) error           { return referenced(err) }

func (a shared) Exec(ctx context.Context, query string, args ...any) (int64, error) {
	res, err := a.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (a shared) Query(ctx context.Context, query string, args ...any) (sqlshared.Rows, error) {
	rows, err := a.db.QueryContext(ctx, query, args...) //nolint:rowserrcheck // the cursor is handed to the caller, which iterates it and checks Err
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (a shared) QueryRow(ctx context.Context, query string, args ...any) sqlshared.Row {
	return sqlRow{a.db.QueryRowContext(ctx, query, args...)}
}

func (a shared) Insert(ctx context.Context, query string, args ...any) (int64, error) {
	res, err := a.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// preparer is the part of *sql.DB, *sql.Tx and *sql.Conn that compiles a
// statement once.
type preparer interface {
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

var _ sqlshared.Preparer = shared{}

// Prepare compiles query on the bound handle; an execer that cannot prepare
// gets a statement that runs each execution unprepared.
func (a shared) Prepare(ctx context.Context, query string) (sqlshared.Stmt, error) {
	p, ok := a.db.(preparer)
	if !ok {
		return unpreparedStmt{a, query}, nil
	}
	st, err := p.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return sqlStmt{st}, nil
}

type sqlStmt struct{ st *sql.Stmt }

func (s sqlStmt) Exec(ctx context.Context, args ...any) error {
	_, err := s.st.ExecContext(ctx, args...)
	return err
}
func (s sqlStmt) Close() error { return s.st.Close() }

type unpreparedStmt struct {
	db    shared
	query string
}

func (s unpreparedStmt) Exec(ctx context.Context, args ...any) error {
	_, err := s.db.Exec(ctx, s.query, args...)
	return err
}
func (unpreparedStmt) Close() error { return nil }

// Transact opens a deferred transaction: the shared stores use it for read
// snapshots too (Compute pins one connection for its session tables), which
// must not hold the write lock for their whole length.
func (a shared) Transact(ctx context.Context, fn func(sqlshared.Execer) error) error {
	return withDeferredTx(ctx, a.db, func(tx execer) error { return fn(shared{db: tx}) })
}

// TransactWrite is Transact for a transaction that writes: it takes the
// write lock at BEGIN (withTx).
func (a shared) TransactWrite(ctx context.Context, fn func(sqlshared.Execer) error) error {
	return withTx(ctx, a.db, func(tx execer) error { return fn(shared{db: tx}) })
}

// RefusesWrites reports the connection's query_only setting, which the
// read-only fallback turns on. Over the pool the answer is read once: the
// setting comes with the DSN and holds for the handle's life, and asking it
// before every statistics read would take a pooled connection each time.
func (a shared) RefusesWrites(ctx context.Context) bool {
	if a.probe != nil {
		if v, ok := a.probe.get(); ok {
			return v
		}
	}
	var on int
	if err := a.db.QueryRowContext(ctx, `PRAGMA query_only`).Scan(&on); err != nil {
		return false
	}
	if a.probe != nil {
		a.probe.set(on != 0)
	}
	return on != 0
}

// writeProbe caches a handle's query_only answer once it has been read.
type writeProbe struct{ state atomic.Int32 } // 0 unknown, 1 writes, 2 refuses

func (p *writeProbe) get() (refuses, known bool) {
	switch p.state.Load() {
	case 1:
		return false, true
	case 2:
		return true, true
	}
	return false, false
}

func (p *writeProbe) set(refuses bool) {
	if refuses {
		p.state.Store(2)
	} else {
		p.state.Store(1)
	}
}

// FillKey names the file the pool reaches: one SQLite file is one tenant, so
// the scope does not tell two fills apart. A transaction has no key.
func (a shared) FillKey(string) (any, bool) {
	sqlDB, ok := a.db.(*sql.DB)
	return sqlDB, ok
}

// LockContention reports whether err is SQLITE_BUSY, extended codes
// included: another writer held the file past busy_timeout, or committed
// since this connection's read began.
func (shared) LockContention(err error) bool {
	var e *sqlitedriver.Error
	return errors.As(err, &e) && e.Code()&0xff == sqliteBusyCode
}

// sqlRow translates database/sql's no-row sentinel to the shared one.
type sqlRow struct{ row *sql.Row }

func (r sqlRow) Scan(dest ...any) error {
	err := r.row.Scan(dest...)
	if errors.Is(err, sql.ErrNoRows) {
		return sqlshared.ErrNoRows
	}
	return err
}
