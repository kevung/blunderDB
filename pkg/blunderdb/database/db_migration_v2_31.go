package database

import "context"

// migrate_2_30_0_to_2_31_0 covers three additions, all created by EnsureSchema
// from the one schema after the chain: match_equity_table and
// analysis.met_digest (ADR-0068), lesson_progress (ADR-0069), and
// move.error_mp. Every column is added NULL, which SQLite records in the
// table definition without touching a row, so the step costs nothing on a
// 15 M-position library. Scoring the existing moves is not part of it: that
// is the resumable MatchStore.ScoreMoves pass, run outside the open.
func (d *Database) migrate_2_30_0_to_2_31_0(context.Context) error {
	return nil
}
