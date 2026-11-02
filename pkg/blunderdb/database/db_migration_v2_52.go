package database

import "context"

// migrate_2_51_0_to_2_52_0 has nothing to alter: the change journal and its
// triggers come from sqlite.EnsureChangeLog, run by EnsureObjectUUIDs after
// the chain, once the uuids of the rows held are in place.
func (d *Database) migrate_2_51_0_to_2_52_0(context.Context) error { return nil }
