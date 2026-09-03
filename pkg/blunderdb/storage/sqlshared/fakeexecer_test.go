package sqlshared

import (
	"context"
	"errors"
	"reflect"
)

// This file gives the B.6 (#174) regression tests a way to simulate "the
// database returned a genuine error mid-scan" without a real corrupted
// SQLite/PostgreSQL file: Execer is already the seam every shared store runs
// against (sqlshared.go), so a fake implementation lets a test fail exactly
// one row of exactly one query and check that the store surfaces an error
// instead of quietly finishing with fewer rows or a lower PR than the data
// actually has.

// fakeDialect answers every Dialect question with the simplest SQLite-shaped
// fact; the SQL text itself is never executed by these fakes; only its
// argument count and the shape of the destinations passed to Scan matter.
type fakeDialect struct{}

func (fakeDialect) Name() string              { return "fake" }
func (fakeDialect) ScopeColumn() string       { return "scope" }
func (fakeDialect) ScopeArg(scope string) any { return scope }
func (fakeDialect) TenantFilter(alias, scope string) (string, []any) {
	return "1=1", nil
}
func (fakeDialect) TenantColumns(scope string) ([]string, []any) { return nil, nil }
func (fakeDialect) Bool(col string, v bool) string {
	if v {
		return col + " = 1"
	}
	return col + " = 0"
}
func (fakeDialect) Bigint(expr string) string       { return expr }
func (fakeDialect) ILike() string                   { return "LIKE" }
func (fakeDialect) LimitOffset(limit, offset int) (string, []any) {
	if limit <= 0 && offset <= 0 {
		return "", nil
	}
	return " LIMIT ? OFFSET ?", []any{limit, offset}
}
func (fakeDialect) TimestampArg() string            { return "?" }
func (fakeDialect) DateText(col string) string      { return col }
func (fakeDialect) TimestampText(col string) string { return col }
func (fakeDialect) Referenced(err error) error      { return err }

// zeroRow is a Row that fills every destination with its zero value and
// never fails — the answer every QueryRow the test does not care about
// should give, so Compute (and friends) can walk past it to the one Query
// call under test.
type zeroRow struct{}

func (zeroRow) Scan(dest ...any) error {
	for _, d := range dest {
		v := reflect.ValueOf(d)
		if v.Kind() == reflect.Ptr && !v.IsNil() {
			v.Elem().Set(reflect.Zero(v.Elem().Type()))
		}
	}
	return nil
}

// emptyRows is a Rows with no rows and no error — the answer every Query the
// test does not care about should give.
type emptyRows struct{}

func (emptyRows) Next() bool             { return false }
func (emptyRows) Scan(dest ...any) error { return nil }
func (emptyRows) Err() error             { return nil }
func (emptyRows) Close() error           { return nil }

// corruptRows yields exactly one row whose Scan always fails — the shape of
// a query that hit one bad row among possibly many others (a NULL where the
// column is not nullable in practice, a value the driver cannot convert to
// the destination type). Before B.6 several call sites turned this into
// "stop accumulating and report what you have so far" rather than an error.
type corruptRows struct{ seen bool }

func (r *corruptRows) Next() bool {
	if r.seen {
		return false
	}
	r.seen = true
	return true
}
func (r *corruptRows) Scan(dest ...any) error { return errors.New("corrupt row") }
func (r *corruptRows) Err() error             { return nil }
func (r *corruptRows) Close() error           { return nil }

// failingRow is a Row whose Scan always fails — the shape of a QueryRow
// call that hit a genuine query/scan failure (a locked database, a dropped
// connection), as opposed to storage.ErrNotFound (no such row), which every
// caller here already handles correctly.
type failingRow struct{}

func (failingRow) Scan(dest ...any) error { return errors.New("simulated query row failure") }

// fakeExecer drives a StatsStore/SearchStore through the Execer seam. Every
// QueryRow returns a zeroRow and every Query returns emptyRows, except:
//   - the queryCallToFail'th Query call (1-indexed, 0 disables it) returns
//     corruptRows (one row whose Scan fails) — a Scan error deep in a
//     multi-row result set;
//   - the queryErrToFail'th Query call (1-indexed, 0 disables it) fails
//     outright, for the sites that swallowed a Query error rather than a
//     Scan error (search_helpers.go);
//   - the queryRowCallToFail'th QueryRow call (1-indexed, 0 disables it)
//     returns failingRow, for the `_ = …QueryRow(...).Scan(...)` sites.
type fakeExecer struct {
	fakeDialect
	queryCalls    int
	queryRowCalls int

	queryCallToFail    int
	queryErrToFail     int
	queryRowCallToFail int
}

func (f *fakeExecer) Exec(ctx context.Context, query string, args ...any) (int64, error) {
	return 0, nil
}

func (f *fakeExecer) Query(ctx context.Context, query string, args ...any) (Rows, error) {
	f.queryCalls++
	if f.queryErrToFail != 0 && f.queryCalls == f.queryErrToFail {
		return nil, errors.New("simulated query failure (locked database)")
	}
	if f.queryCallToFail != 0 && f.queryCalls == f.queryCallToFail {
		return &corruptRows{}, nil
	}
	return emptyRows{}, nil
}

func (f *fakeExecer) QueryRow(ctx context.Context, query string, args ...any) Row {
	f.queryRowCalls++
	if f.queryRowCallToFail != 0 && f.queryRowCalls == f.queryRowCallToFail {
		return failingRow{}
	}
	return zeroRow{}
}

func (f *fakeExecer) Insert(ctx context.Context, query string, args ...any) (int64, error) {
	return 0, nil
}

func (f *fakeExecer) Transact(ctx context.Context, fn func(Execer) error) error {
	return fn(f)
}
