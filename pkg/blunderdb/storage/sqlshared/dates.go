package sqlshared

import (
	"context"
	"fmt"
	"strings"
)

// KeepDates gives the row of table whose keyCol is key the dates its source
// row bore, as TimestampText renders them: an import carries a library's
// rows with their history instead of stamping them with the day of the copy
// (ADR-0007). An empty date leaves its column as the insert set it; a table
// without updated_at is called with updated "". A key that names no row of
// the scope changes nothing and is no error: there is nothing to date.
//
// table and keyCol come from the stores, never from a caller's input.
func KeepDates(ctx context.Context, db Execer, scope, table, keyCol string, key any, created, updated string) error {
	var sets []string
	var args []any
	if created != "" {
		sets, args = append(sets, "created_at = "+db.TimestampTextArg()), append(args, created)
	}
	if updated != "" {
		sets, args = append(sets, "updated_at = "+db.TimestampTextArg()), append(args, updated)
	}
	if len(sets) == 0 {
		return nil
	}
	tenant, targs := db.TenantFilter("", scope)
	args = append(append(args, key), targs...)
	if _, err := db.Exec(ctx, `UPDATE `+table+` SET `+strings.Join(sets, ", ")+
		` WHERE `+keyCol+` = ? AND `+tenant, args...); err != nil {
		return fmt.Errorf("%s: keep dates of %s %v: %w", db.Name(), table, key, err)
	}
	return nil
}
