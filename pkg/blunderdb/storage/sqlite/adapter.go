package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// shared adapts the backend's execer (a *sql.DB or a *sql.Tx) to the
// sqlshared.Execer the backend-agnostic stores run against, and answers the
// dialect questions for SQLite: '?' placeholders as-is, no tenant column on the
// domain tables, text dates and integer aggregates that need no cast.
type shared struct{ db execer }

var _ sqlshared.Execer = shared{}

// shared returns the binder's execer wrapped for the shared stores.
func (b binder) shared() sqlshared.Execer { return shared(b) }

func (shared) Name() string                             { return "sqlite" }
func (shared) ScopeColumn() string                      { return "scope" }
func (shared) ScopeArg(scope string) any                { return scope }
func (shared) TenantFilter(_, _ string) (string, []any) { return "1=1", nil }
func (shared) TenantColumns(string) ([]string, []any)   { return nil, nil }
func (shared) Bool(col string, v bool) string {
	if v {
		return col + " = 1"
	}
	return col + " = 0"
}
func (shared) Bigint(expr string) string       { return expr }
func (shared) ILike() string                   { return "LIKE" }
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
func (shared) TimestampArg() string            { return "?" }
func (shared) DateText(col string) string      { return "COALESCE(" + col + ",'')" }
func (shared) TimestampText(col string) string { return "COALESCE(" + col + ",'')" }
func (shared) Referenced(err error) error      { return referenced(err) }

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

func (a shared) Transact(ctx context.Context, fn func(sqlshared.Execer) error) error {
	return withTx(ctx, a.db, func(tx execer) error { return fn(shared{tx}) })
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
