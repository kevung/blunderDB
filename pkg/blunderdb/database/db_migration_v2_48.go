package database

import "context"

// migrate_2_47_0_to_2_48_0 gives match_stats each seat's MWC loss and its
// per-game spread, which the badges and the players table read instead of
// the decisions. The rows written before carry NULL in mwc_games: the
// older-shape repair at the end of the chain drops them and the next reader
// recomputes them.
func (d *Database) migrate_2_47_0_to_2_48_0(context.Context) error {
	present, err := d.columnExists("match_stats", "match_id")
	if err != nil || !present {
		return err
	}
	for _, col := range []string{"mwc_loss REAL", "mwc_games INTEGER", "mwc_game_ss REAL"} {
		if err := d.addColumn("match_stats", col); err != nil {
			return err
		}
	}
	return nil
}
