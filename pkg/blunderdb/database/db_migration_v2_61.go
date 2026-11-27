package database

import "context"

// migrate_2_60_0_to_2_61_0 gives an analysis its best_move column
// (engine.BestMove), which a move-pattern search narrows by. The rows stored
// before keep it NULL, "not derived yet": the search lets them through and
// decides on the blob, so nothing is owed. No reader floor: a row an older
// binary inserts is one more NULL. One it re-analyses keeps the best move
// read before, which a search can miss if the new analysis changed it.
func (d *Database) migrate_2_60_0_to_2_61_0(context.Context) error {
	return d.addColumn("analysis", "best_move TEXT")
}
