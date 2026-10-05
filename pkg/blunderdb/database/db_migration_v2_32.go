package database

import "context"

// migrate_2_31_0_to_2_32_0 adds the two tables of the Duel (ADR-0072): duel,
// the drafts of the matches being played here, and match_origin, how a Match
// played here came to be. Both are new and created by EnsureSchema from the
// one schema after the chain; no row of an existing library changes.
func (d *Database) migrate_2_31_0_to_2_32_0(context.Context) error {
	return nil
}
