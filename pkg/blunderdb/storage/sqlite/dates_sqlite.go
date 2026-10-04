package sqlite

import "strings"

// UnixFromMatchDateSQL renders the SQL expression that turns a match.match_date
// value into the Unix seconds position.match_date stores (ADR-0071).
//
// match.match_date keeps the text the driver writes for a time.Time, Go's
// String form "2006-01-02 15:04:05[.fffffffff] -0700 MST". The first 19
// characters are the wall clock, which unixepoch reads as UTC; the zone offset
// after the first space past them is then subtracted, so a match dated in
// another zone lands on the same instant Go would compute. A value without an
// offset (or a bare date) reads as UTC, and an integer passes through: a
// position date derived from either kind of match row compares as one scale.
func UnixFromMatchDateSQL(col string) string {
	return strings.ReplaceAll(`(CASE
	WHEN typeof(@c) = 'integer' THEN @c
	WHEN @c IS NULL THEN NULL
	ELSE unixepoch(substr(@c, 1, 19)) - (
		CASE substr(substr(@c, 20), instr(substr(@c, 20), ' ') + 1, 1)
			WHEN '+' THEN 1 WHEN '-' THEN -1 ELSE 0 END
		* (CAST(substr(substr(@c, 20), instr(substr(@c, 20), ' ') + 2, 2) AS INTEGER) * 3600
		 + CAST(substr(substr(@c, 20), instr(substr(@c, 20), ' ') + 4, 2) AS INTEGER) * 60))
	END)`, "@c", col)
}
