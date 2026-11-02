package database

import "context"

// migrate_2_52_0_to_2_53_0 gives a position its public_id: free text the
// engine keeps and returns, never reads. The column stays NULL until a caller
// sets it, so no row is rewritten and the change journal is not touched.
func (d *Database) migrate_2_52_0_to_2_53_0(context.Context) error {
	return d.addColumn("position", "public_id TEXT")
}
