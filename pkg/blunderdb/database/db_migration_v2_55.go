package database

import "context"

// migrate_2_54_0_to_2_55_0 adds the derivation of a position. The table itself
// is created by EnsureSchema from the one schema, after the chain, like every
// table added since 2.31.0; nothing to rewrite here.
func (d *Database) migrate_2_54_0_to_2_55_0(context.Context) error { return nil }
