// Package sqlshared holds the store implementations that are the same SQL on
// both backends, written once against a small adapter instead of twice against
// *sql.DB and pgx.
//
// The two backends (storage/sqlite, storage/postgres) had grown sixteen pairs
// of files that were identical modulo the driver's method names
// (QueryRowContext vs QueryRow), the placeholder style ('?' vs '$N'), a few
// type casts and the tenant column. Each pair drifted a little with every
// change — a comment shortened on one side, a COALESCE added on the other. The
// families where the SQL is genuinely the same now live here, and each backend
// keeps only what really differs: how to reach the database (Execer) and the
// handful of dialect facts the shared SQL has to ask about (Dialect).
//
// What stays in the backends: the families whose statements differ in shape
// (RETURNING, ON CONFLICT targets, tenant_id on every insert — positions,
// analyses, matches, collections, tournaments) and the one-off methods noted
// on each shared store. anki (B.14, #182) is the shared AnkiStore here save
// for Forecast — its day-offset bucketing is genuine date-arithmetic
// divergence (SQLite has no DATE type and computes it through julianday(),
// PostgreSQL natively) — which each backend still writes itself, embedding
// AnkiStore and shadowing that one method (the stats_sqlite.go/
// stats_postgres.go precedent for DateRange).
//
// Every query in this package is written with '?' placeholders; the PostgreSQL
// adapter rebinds them to '$N' before execution. No query here contains a
// string literal with a '?' in it, so the sequential substitution is safe.
package sqlshared

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrNoRows is what Row.Scan returns when the query matched nothing. The
// adapters translate their driver's sentinel (sql.ErrNoRows, pgx.ErrNoRows) to
// this one so the shared stores test a single error.
var ErrNoRows = errors.New("sqlshared: no rows in result set")

// Rows is the cursor returned by Execer.Query — the database/sql shape, which
// pgx.Rows also satisfies once Close is given an error return.
type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close() error
}

// Row is a single-row result; Scan returns ErrNoRows when nothing matched.
type Row interface {
	Scan(dest ...any) error
}

// Execer is the query surface a shared store runs against. Both backends
// implement it over their autocommit handle and over an open transaction, so a
// store works in either binding. The embedded Dialect answers the questions
// the shared SQL has to ask about the backend.
type Execer interface {
	Dialect

	// Exec runs a statement and reports the number of rows it affected.
	Exec(ctx context.Context, query string, args ...any) (rowsAffected int64, err error)
	// Query runs a statement that returns rows.
	Query(ctx context.Context, query string, args ...any) (Rows, error)
	// QueryRow runs a statement expected to return at most one row.
	QueryRow(ctx context.Context, query string, args ...any) Row
	// Insert runs an INSERT (written without RETURNING) and reports the id of
	// the new row: LastInsertId on SQLite, RETURNING id on PostgreSQL.
	Insert(ctx context.Context, query string, args ...any) (id int64, err error)
	// Transact runs fn atomically. Bound to an autocommit handle it opens a
	// transaction around fn; bound to a transaction already, it lets that
	// transaction (or a savepoint under it) provide the atomicity.
	Transact(ctx context.Context, fn func(Execer) error) error
}

// Dialect is the set of facts in which the two backends' SQL actually differs.
// It is deliberately small: anything that can be written the same way on both
// (ON CONFLICT upserts, CAST(x AS INTEGER), CURRENT_TIMESTAMP) is written the
// same way in the shared stores rather than abstracted here.
type Dialect interface {
	// Name is the backend's name, used as the prefix of every error message
	// ("sqlite: …", "postgres: …") so a log line still says which store failed.
	Name() string

	// ScopeColumn and ScopeArg describe the per-scope column of the tables
	// that carry one in both schemas (command_history, search_history,
	// filter_library): the SQLite schema stores the scope string in a `scope`
	// column, PostgreSQL the numeric tenant in `tenant_id`.
	ScopeColumn() string
	ScopeArg(scope string) any

	// TenantFilter renders the predicate that confines a domain table
	// (position, match, comment, …) to the scope's tenant, qualified by alias
	// when one is given. The SQLite schema has no tenant column on those
	// tables — a private database is its own single tenant — so it returns
	// the constant TRUE and no argument; PostgreSQL returns
	// "alias.tenant_id = ?" with the tenant id. Callers put the predicate
	// first in the WHERE clause and its arguments first in the argument list,
	// so the '?' placeholders line up after rebinding.
	TenantFilter(alias, scope string) (pred string, args []any)

	// TenantColumns is TenantFilter's INSERT counterpart: the columns (and
	// their values) an insert into a domain table must carry to land in the
	// scope's tenant — none on SQLite, tenant_id on PostgreSQL. Callers put
	// them first in the column list and the argument list.
	TenantColumns(scope string) (cols []string, args []any)

	// Bool renders the predicate "col is v". The SQLite schema stores its
	// flags as INTEGER 0/1 and declares partial indexes on `col = 1`, which
	// the planner only matches textually, so the literal has to stay 1/0
	// there; PostgreSQL's BOOLEAN columns take TRUE/FALSE.
	Bool(col string, v bool) string

	// BoolAsInt renders a boolean-shaped column in a SELECT list so the same
	// Go int destination scans it on both backends: unchanged on SQLite
	// (already INTEGER 0/1), a CASE on PostgreSQL (native BOOLEAN).
	BoolAsInt(col string) string

	// BoolArg converts v into the argument a boolean-shaped column's INSERT/
	// UPDATE wants bound: SQLite stores its flags as INTEGER 0/1 (see Bool),
	// so this returns 0/1; PostgreSQL's BOOLEAN column takes v unchanged.
	BoolArg(v bool) any

	// Bigint wraps an aggregate so it scans into an int64: SUM over BIGINT is
	// NUMERIC in PostgreSQL (cast back), plain integer in SQLite (unchanged).
	Bigint(expr string) string

	// ILike is the case-insensitive LIKE operator: LIKE in SQLite (already
	// case-insensitive for ASCII), ILIKE in PostgreSQL.
	ILike() string

	// LimitOffset renders a ListOpts as the trailing clause of a SELECT
	// already carrying its ORDER BY, and the placeholder arguments it needs —
	// "", nil when both are zero (no bound, today's behaviour). PostgreSQL
	// accepts a bare OFFSET with no LIMIT; SQLite's grammar requires a LIMIT
	// whenever OFFSET is given, so an offset with no limit asks SQLite for an
	// explicit unbounded one (LIMIT -1) instead.
	LimitOffset(limit, offset int) (clause string, args []any)

	// TimestampArg is the placeholder for a match_date bound as a string:
	// "?" in SQLite (match_date is TEXT), "?::timestamptz" in PostgreSQL.
	TimestampArg() string

	// DateText renders a match_date column as a text date (YYYY-MM-DD…, '' for
	// NULL) and TimestampText a created_at/modified_at column as text
	// ('' for NULL). SQLite stores both as text already; PostgreSQL formats
	// its TIMESTAMPTZ in UTC.
	DateText(col string) string
	TimestampText(col string) string

	// Referenced maps the backend's FOREIGN KEY violation onto
	// storage.ErrNotFound (the row the caller pointed at does not exist) and
	// passes any other error through.
	Referenced(err error) error
}

// Rebind rewrites a query built with positional '?' placeholders into the
// PostgreSQL '$1, $2, …' form. The queries in this package contain no string
// literal with a '?', so a straight sequential substitution is correct.
func Rebind(query string) string {
	if !strings.Contains(query, "?") {
		return query
	}
	var b strings.Builder
	b.Grow(len(query) + 16)
	n := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
			continue
		}
		b.WriteByte(query[i])
	}
	return b.String()
}

// Placeholders returns n comma-separated '?' placeholders for an IN list.
func Placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

// errf wraps err with the backend's name and what was being done, so every
// error out of a shared store reads "sqlite: save filter "x": …".
func errf(d Dialect, what string, err error) error {
	return fmt.Errorf("%s: %s: %w", d.Name(), what, err)
}
