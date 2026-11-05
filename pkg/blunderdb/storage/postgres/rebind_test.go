package postgres

import "testing"

func TestTypedRebindTypesOnlyComparedIntegers(t *testing.T) {
	cases := []struct {
		query string
		args  []any
		want  string
	}{
		{"SELECT 1 WHERE substr(key, 1, ?) = ?", []any{4, "mnt:"}, "SELECT 1 WHERE substr(key, 1, $1) = $2"},
		{"WHERE cube = ? AND a <> ? AND b >= ?", []any{40000, 1, 2}, "WHERE cube = $1::bigint AND a <> $2::bigint AND b >= $3::bigint"},
		{"WHERE ? <= score", []any{9}, "WHERE $1::bigint <= score"},
		{"WHERE id IN (?,?) AND x = ?", []any{1, 2, "s"}, "WHERE id IN ($1::bigint,$2::bigint) AND x = $3"},
		{"WHERE f(?, ?) AND id IN (SELECT g(?) FROM t WHERE y = ?)", []any{1, 2, 3, 4}, "WHERE f($1, $2) AND id IN (SELECT g($3) FROM t WHERE y = $4::bigint)"},
		{"WHERE s BETWEEN ? AND ? AND ok", []any{0, 5}, "WHERE s BETWEEN $1::bigint AND $2::bigint AND ok"},
		{"INSERT INTO t (a, b) VALUES (?, ?) LIMIT ?", []any{1, 2, 3}, "INSERT INTO t (a, b) VALUES ($1, $2) LIMIT $3"},
		{"WHERE a = ?::smallint", []any{1}, "WHERE a = $1::smallint"},
	}
	for _, c := range cases {
		if got := typedRebind(c.query, c.args); got != c.want {
			t.Errorf("typedRebind(%q)\n got %q\nwant %q", c.query, got, c.want)
		}
	}
}
