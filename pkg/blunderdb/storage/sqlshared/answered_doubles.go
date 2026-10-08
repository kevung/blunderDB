package sqlshared

import (
	"context"
	"fmt"
)

// answerCodesSQL lists the fixed codes of a take or a pass.
var answerCodesSQL = fmt.Sprint(fixedActionCode("Take")) + `, ` + fmt.Sprint(fixedActionCode("Pass"))

// AnsweredOwnedCubeMovesSQL selects (move, game, position) of every take or
// pass recorded on a position whose turned cube the answerer owns. Such a
// position is the answerer's own redouble decision on that board — same
// Zobrist hash, same analysis — where a reply is recorded with the turned
// cube held by no one, as importers record it. The rows decide, not the
// match's provenance (a transcription rewrites an imported match keeping its
// file): an importer that carries no separate response lets the take fall
// back onto the doubler's own position, which the double of the same game
// stands on too, and that row is right. That fallback can also leave a take
// alone on the doubler's own row when it owned the cube before redoubling,
// which looks the same; there the take is the doubler's record, while an
// answer recorded by a transcription follows the opponent's own Double move.
// Aliases mv, g, m, p; the caller
// appends its tenant predicate.
var AnsweredOwnedCubeMovesSQL = `SELECT mv.id, mv.game_id, mv.position_id FROM move mv
	JOIN game g ON g.id = mv.game_id
	JOIN match m ON m.id = g.match_id
	JOIN position p ON p.id = mv.position_id
	WHERE ` + ActionCodeOrEmptySQL("mv.cube_action") + ` IN (` + answerCodesSQL + `)
	  AND p.decision_type = 1 AND p.cube_value > 0 AND p.cube_owner = p.player_on_roll
	  AND NOT EXISTS (SELECT 1 FROM move d WHERE d.game_id = mv.game_id AND d.position_id = mv.position_id
	      AND ` + ActionCodeOrEmptySQL("d.cube_action") + ` NOT IN (` + answerCodesSQL + `))
	  AND EXISTS (SELECT 1 FROM move o WHERE o.game_id = mv.game_id AND o.move_number = mv.move_number - 1
	      AND ` + ActionIsSQL("o.cube_action", "Double") + ` AND o.player <> mv.player)`

// DropGammonNetResponseAnalyses deletes every gammonNet analysis stored on a
// take/pass position (a turned cube held by no one): gammonNet once scored
// such a row as the answerer's centred-cube decision, a verdict its version
// label does not tell apart from the doubler's one it now gives, so the row
// is emptied for the next analysis to fill. Other engines' verdicts are the
// doubler's already and stay. The moves on those rows lose their error and
// the match statistics that summarised them are dropped. Every scope the
// handle sees; it returns how many analyses went.
func DropGammonNetResponseAnalyses(ctx context.Context, db Execer) (int, error) {
	rows, err := db.Query(ctx, `SELECT a.position_id FROM analysis a JOIN position p ON p.id = a.position_id
		WHERE a.analysis_engine LIKE 'gammonNet%' AND p.decision_type = 1 AND p.cube_value > 0 AND p.cube_owner = -1
		ORDER BY a.position_id`)
	if err != nil {
		return 0, errf(db, "list gammonNet response analyses", err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, errf(db, "list gammonNet response analyses", err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, errf(db, "list gammonNet response analyses", err)
	}
	if len(ids) == 0 {
		return 0, nil
	}
	err = db.Transact(ctx, func(tx Execer) error {
		for start := 0; start < len(ids); start += playedDecisionChunk {
			part := int64Args(ids[start:min(start+playedDecisionChunk, len(ids))])
			in := `(` + Placeholders(len(part)) + `)`
			if _, err := tx.Exec(ctx, `DELETE FROM match_stats WHERE match_id IN
				(SELECT g.match_id FROM move mv JOIN game g ON g.id = mv.game_id WHERE mv.position_id IN `+in+`)`, part...); err != nil {
				return fmt.Errorf("invalidate match stats of response analyses: %w", err)
			}
			if _, err := tx.Exec(ctx, `DELETE FROM analysis WHERE position_id IN `+in, part...); err != nil {
				return fmt.Errorf("delete response analyses: %w", err)
			}
		}
		_, err := RescorePlayedDecisionsOf(ctx, tx, ids)
		return err
	})
	if err != nil {
		return 0, errf(db, "drop gammonNet response analyses", err)
	}
	return len(ids), nil
}
