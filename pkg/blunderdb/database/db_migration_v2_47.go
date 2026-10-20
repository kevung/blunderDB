package database

import "context"

// migrate_2_46_0_to_2_47_0 gives a comment room for its producer's signature
// (ADR-0099). Every comment written before is unsigned: NULL.
func (d *Database) migrate_2_46_0_to_2_47_0(context.Context) error {
	return d.addColumn("comment", "signature TEXT")
}
