package database

import "context"

// migrate_2_42_0_to_2_43_0 dates the visits: match.last_visited_at is set
// whenever the last visited position is. A match visited before keeps NULL,
// as the instant was never kept; it sorts after the dated visits.
func (d *Database) migrate_2_42_0_to_2_43_0(context.Context) error {
	return d.addColumn("match", "last_visited_at DATETIME")
}
