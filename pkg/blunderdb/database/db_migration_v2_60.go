package database

import "context"

// migrate_2_59_0_to_2_60_0 gives a tournament its origin (domain.Tournament.
// Origin), a free text naming where it was played. A tournament stored before
// gets an empty one, unknown, rather than "imported": a tournament created by hand in
// the desktop was not imported, and nothing in the row tells the two apart.
// The step raises no reader floor: an older binary reads the table and never
// writes the column. Through a synchronisation its tournament payload carries
// no origin and leaves the stored one alone; only a copy an older binary makes
// itself, such as its whole-library export, leaves the origin behind.
func (d *Database) migrate_2_59_0_to_2_60_0(context.Context) error {
	return d.addColumn("tournament", "origin TEXT NOT NULL DEFAULT ''")
}
