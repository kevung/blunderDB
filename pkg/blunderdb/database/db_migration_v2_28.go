package database

import "context"

// migrate_2_27_0_to_2_28_0 covers the properties of a table and the rooms of
// an event (ADR-0058): table_setting and tournament.rencontre_rooms. Both are
// new and empty on an existing library, so EnsureSchema creates them from the
// one schema; the step keeps the chain continuous
// (TestMigrationSteps_ContinuousChain).
func (d *Database) migrate_2_27_0_to_2_28_0(context.Context) error {
	return nil
}
