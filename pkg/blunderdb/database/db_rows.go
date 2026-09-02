package database

import "database/sql"

// rowQuerier is the one method the cursor helpers below need; *sql.DB and
// *sql.Tx both satisfy it, so a helper works inside or outside a transaction.
type rowQuerier interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

// forEachRow runs query and hands every row to fn, closing the cursor before
// it returns — whether the rows ran out, fn failed, or the iteration did. fn
// returning nil moves on to the next row.
//
// It is the loop-safe shape of `rows, err := Query(…); for rows.Next() {…}`:
// a cursor opened inside a loop cannot be closed by a function-level defer
// without keeping every one of them open until the function returns, and the
// manual Close on each early-return path is exactly what gets forgotten.
func forEachRow(q rowQuerier, query string, args []any, fn func(*sql.Rows) error) error {
	rows, err := q.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := fn(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

// queryInt64s runs a single-column query and returns every value, with the
// cursor closed before it returns. Most callers read a list of ids and then
// write on the same connection: the cursor has to be gone by then, and a
// defer in the caller would keep it open across those writes.
func queryInt64s(q rowQuerier, query string, args ...any) ([]int64, error) {
	var out []int64
	err := forEachRow(q, query, args, func(rows *sql.Rows) error {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return err
		}
		out = append(out, v)
		return nil
	})
	return out, err
}
