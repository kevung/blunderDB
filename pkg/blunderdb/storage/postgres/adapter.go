package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// shared adapts the backend's execer (a *pgxpool.Pool or a pgx.Tx) to the
// sqlshared.Execer the backend-agnostic stores run against, and answers the
// dialect questions for PostgreSQL: '?' placeholders rebound to '$N', every
// domain table confined to the scope's tenant_id, BIGINT aggregates cast back
// from NUMERIC, TIMESTAMPTZ dates formatted in UTC.
type shared struct{ db execer }

var _ sqlshared.Execer = shared{}

// shared returns the binder's execer wrapped for the shared stores.
func (b binder) shared() sqlshared.Execer { return shared(b) }

func (shared) Name() string              { return "postgres" }
func (shared) ScopeColumn() string       { return "tenant_id" }
func (shared) ScopeArg(scope string) any { return tenantID(scope) }
func (shared) TenantFilter(alias, scope string) (string, []any) {
	col := "tenant_id"
	if alias != "" {
		col = alias + ".tenant_id"
	}
	return col + " = ?", []any{tenantID(scope)}
}
func (shared) Bigint(expr string) string { return "CAST(" + expr + " AS BIGINT)" }
func (shared) ILike() string             { return "ILIKE" }
func (shared) TimestampArg() string      { return "?::timestamptz" }
func (shared) DateText(col string) string {
	return "COALESCE(TO_CHAR(" + col + " AT TIME ZONE 'UTC','YYYY-MM-DD'),'')"
}
func (shared) TimestampText(col string) string {
	return "COALESCE(TO_CHAR(" + col + " AT TIME ZONE 'UTC','YYYY-MM-DD HH24:MI:SS'),'')"
}
func (shared) Referenced(err error) error { return referenced(err) }

func (a shared) Exec(ctx context.Context, query string, args ...any) (int64, error) {
	tag, err := a.db.Exec(ctx, sqlshared.Rebind(query), args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (a shared) Query(ctx context.Context, query string, args ...any) (sqlshared.Rows, error) {
	rows, err := a.db.Query(ctx, sqlshared.Rebind(query), args...) //nolint:rowserrcheck // the cursor is handed to the caller, which iterates it and checks Err
	if err != nil {
		return nil, err
	}
	return pgRows{rows}, nil
}

func (a shared) QueryRow(ctx context.Context, query string, args ...any) sqlshared.Row {
	return pgRow{a.db.QueryRow(ctx, sqlshared.Rebind(query), args...)}
}

func (a shared) Insert(ctx context.Context, query string, args ...any) (int64, error) {
	var id int64
	if err := a.db.QueryRow(ctx, sqlshared.Rebind(query)+" RETURNING id", args...).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (a shared) Transact(ctx context.Context, fn func(sqlshared.Execer) error) error {
	return withTx(ctx, a.db, func(tx execer) error { return fn(shared{tx}) })
}

// pgRows gives pgx.Rows the Close() error signature of database/sql.
type pgRows struct{ pgx.Rows }

func (r pgRows) Close() error {
	r.Rows.Close()
	return nil
}

// pgRow translates pgx's no-row sentinel to the shared one.
type pgRow struct{ row pgx.Row }

func (r pgRow) Scan(dest ...any) error {
	err := r.row.Scan(dest...)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlshared.ErrNoRows
	}
	return err
}
