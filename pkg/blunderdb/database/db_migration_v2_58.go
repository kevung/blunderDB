package database

import (
	"context"
	"fmt"
)

// matchResultBackfillSQL computes the result of every match stored before
// it had one, from its last game, as domain.MatchResultOf does at import.
const matchResultBackfillSQL = `UPDATE match SET
	final_score_1 = (SELECT COALESCE(g.initial_score_1, 0) + CASE WHEN g.winner = 1 THEN COALESCE(g.points_won, 0) ELSE 0 END
	                   FROM game g WHERE g.match_id = match.id ORDER BY g.game_number DESC, g.id DESC LIMIT 1),
	final_score_2 = (SELECT COALESCE(g.initial_score_2, 0) + CASE WHEN g.winner = -1 THEN COALESCE(g.points_won, 0) ELSE 0 END
	                   FROM game g WHERE g.match_id = match.id ORDER BY g.game_number DESC, g.id DESC LIMIT 1)
 WHERE final_score_1 IS NULL`

const matchWinnerBackfillSQL = `UPDATE match SET winner = CASE
	WHEN COALESCE(match_length, 0) > 0 AND final_score_1 >= match_length THEN 1
	WHEN COALESCE(match_length, 0) > 0 AND final_score_2 >= match_length THEN -1
	ELSE 0 END
 WHERE final_score_1 IS NOT NULL`

// migrate_2_57_0_to_2_58_0 stores the result of a match (final score and
// winner) beside it, so a list of matches shows it without reading the games.
func (d *Database) migrate_2_57_0_to_2_58_0(context.Context) error {
	for _, col := range []string{"final_score_1 INTEGER", "final_score_2 INTEGER", "winner INTEGER NOT NULL DEFAULT 0"} {
		if err := d.addColumn("match", col); err != nil {
			return err
		}
	}
	for _, stmt := range []string{matchResultBackfillSQL, matchWinnerBackfillSQL} {
		if _, err := d.db.Exec(stmt); err != nil {
			return fmt.Errorf("computing match results: %w", err)
		}
	}
	return nil
}
