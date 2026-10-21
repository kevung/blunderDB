package sqlshared

// TrailingScan is a row whose Scan also fills Extra from the columns that
// follow the ones the caller asks for: a row scanner written for one SELECT
// list then reads a query that appends columns to it, without a second copy
// of that scanner.
type TrailingScan struct {
	Row   interface{ Scan(dest ...any) error }
	Extra []any
}

// Scan scans dest then Extra, in that column order.
func (t TrailingScan) Scan(dest ...any) error {
	return t.Row.Scan(append(dest, t.Extra...)...)
}
