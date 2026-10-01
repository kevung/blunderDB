package database

import "context"

// migrate_2_26_0_to_2_27_0 covers transcription.revision, the version a
// transcription gesture names (ADR-0057 rule 4). EnsureSchema adds the column
// with its default of 1, the revision a fresh insert starts at; the step keeps
// the chain continuous (TestMigrationSteps_ContinuousChain).
func (d *Database) migrate_2_26_0_to_2_27_0(context.Context) error {
	return nil
}
