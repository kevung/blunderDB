package database

import "context"

// migrate_2_28_0_to_2_29_0 covers the Lessons (ADR-0066): lesson and
// lesson_step. Both are new and empty on an existing library, so EnsureSchema
// creates them from the one schema; the step keeps the chain continuous
// (TestMigrationSteps_ContinuousChain).
func (d *Database) migrate_2_28_0_to_2_29_0(context.Context) error {
	return nil
}
