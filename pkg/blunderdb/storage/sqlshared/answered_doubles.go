package sqlshared

import "fmt"

// AnsweredOwnedCubeMovesSQL selects (move, game, position) of every take or
// pass a transcribed or duelled match recorded on a position whose turned
// cube the answerer owns. Such a position is the answerer's own redouble
// decision on that board — same Zobrist hash, same analysis — where a reply
// is recorded with the turned cube held by no one, as importers record it.
// An imported match is left out: its take/pass falls back onto the doubler's
// own position when the file carries no separate response, and that row is
// right. Aliases mv, g, m, p; the caller appends its tenant predicate.
var AnsweredOwnedCubeMovesSQL = `SELECT mv.id, mv.game_id, mv.position_id FROM move mv
	JOIN game g ON g.id = mv.game_id
	JOIN match m ON m.id = g.match_id
	JOIN position p ON p.id = mv.position_id
	WHERE ` + ActionCodeOrEmptySQL("mv.cube_action") + ` IN (` +
	fmt.Sprint(fixedActionCode("Take")) + `, ` + fmt.Sprint(fixedActionCode("Pass")) + `)
	  AND COALESCE(m.file_path, '') = '' AND m.import_batch_id IS NULL
	  AND p.decision_type = 1 AND p.cube_value > 0 AND p.cube_owner = p.player_on_roll`
