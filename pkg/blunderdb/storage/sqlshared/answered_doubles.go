package sqlshared

import (
	"context"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
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
// Aliases mv, g, m, p, each joined within its tenant; the caller appends
// its tenant predicate.
func AnsweredOwnedCubeMovesSQL(d Dialect) string {
	return `SELECT mv.id, mv.game_id, mv.position_id FROM move mv
	JOIN game g ON g.id = mv.game_id` + andSameTenant(d, "g", "mv") + `
	JOIN match m ON m.id = g.match_id` + andSameTenant(d, "m", "g") + `
	JOIN position p ON p.id = mv.position_id` + andSameTenant(d, "p", "mv") + `
	WHERE ` + ActionCodeOrEmptySQL("mv.cube_action") + ` IN (` + answerCodesSQL + `)
	  AND p.decision_type = 1 AND p.cube_value > 0 AND p.cube_owner = p.player_on_roll
	  AND NOT EXISTS (SELECT 1 FROM move d WHERE d.game_id = mv.game_id AND d.position_id = mv.position_id
	      AND ` + ActionCodeOrEmptySQL("d.cube_action") + ` NOT IN (` + answerCodesSQL + `))
	  AND EXISTS (SELECT 1 FROM move o WHERE o.game_id = mv.game_id AND o.move_number = mv.move_number - 1
	      AND ` + ActionIsSQL("o.cube_action", "Double") + ` AND o.player <> mv.player)`
}

// DropGammonNetResponseAnalyses deletes every gammonNet analysis stored on a
// take/pass position (a turned cube held by no one) whose verdict is the
// answerer's centred-cube decision rather than the doubler's: such a verdict
// reads the take against the wrong equities, and its version label cannot
// tell it from a right one, so the row is emptied for the next analysis to
// fill. The caller runs it once per library, before any right verdict can
// exist. Other engines' verdicts are the doubler's and stay. The moves on
// those rows lose their error and the match statistics that summarised them
// are dropped. One scope; it returns how many analyses went.
func DropGammonNetResponseAnalyses(ctx context.Context, db Execer, scope string) (int, error) {
	tenant, targs := db.TenantFilter("a", scope)
	// A row whose provenance is not derived yet is read from its blob, so
	// the drop does not depend on the provenance pass having run first.
	rows, err := db.Query(ctx, `SELECT a.position_id, a.analysis_engine IS NULL, CASE WHEN a.analysis_engine IS NULL THEN a.data END
		FROM analysis a JOIN position p ON p.id = a.position_id`+andSameTenant(db, "p", "a")+`
		WHERE `+tenant+` AND (a.analysis_engine LIKE 'gammonNet%' OR a.analysis_engine IS NULL)
		  AND p.decision_type = 1 AND p.cube_value > 0 AND p.cube_owner = -1
		ORDER BY a.position_id`, targs...)
	if err != nil {
		return 0, errf(db, "list gammonNet response analyses", err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		var underived bool
		var data []byte
		if err := rows.Scan(&id, &underived, &data); err != nil {
			rows.Close()
			return 0, errf(db, "list gammonNet response analyses", err)
		}
		if underived {
			a, err := engine.DecodeAnalysisFromStorage(data)
			if err != nil {
				continue
			}
			if label, _, _ := engine.AnalysisProvenance(&a); !strings.HasPrefix(label, "gammonNet") {
				continue
			}
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
			chunk := ids[start:min(start+playedDecisionChunk, len(ids))]
			if err := InvalidateMatchStatsOfPositions(ctx, tx, scope, chunk); err != nil {
				return fmt.Errorf("invalidate match stats of response analyses: %w", err)
			}
			analysisTenant, aargs := tx.TenantFilter("", scope)
			if _, err := tx.Exec(ctx, `DELETE FROM analysis WHERE `+analysisTenant+` AND position_id IN (`+Placeholders(len(chunk))+`)`,
				append(append([]any{}, aargs...), int64Args(chunk)...)...); err != nil {
				return fmt.Errorf("delete response analyses: %w", err)
			}
		}
		_, err := RescorePlayedDecisionsOf(ctx, tx, scope, ids)
		return err
	})
	if err != nil {
		return 0, errf(db, "drop gammonNet response analyses", err)
	}
	return len(ids), nil
}

// DropOwnedCubeAnswerMatchStats deletes the statistics of every match of
// scope holding a take or a pass on an owned cube: the doubler's own row,
// where an .xg file without its raw cube segment leaves the take. Those
// statistics converted its MWC loss at half the cube it stands at; they are
// recomputed from the rows.
func DropOwnedCubeAnswerMatchStats(ctx context.Context, db Execer, scope string) error {
	tenant, args := db.TenantFilter("", scope)
	mvTenant, mvArgs := db.TenantFilter("mv", scope)
	_, err := db.Exec(ctx, `DELETE FROM match_stats WHERE `+tenant+` AND match_id IN (
	SELECT g.match_id FROM move mv JOIN game g ON g.id = mv.game_id`+andSameTenant(db, "g", "mv")+`
	JOIN position p ON p.id = mv.position_id`+andSameTenant(db, "p", "mv")+`
	WHERE `+mvTenant+` AND `+ActionCodeOrEmptySQL("mv.cube_action")+` IN (`+answerCodesSQL+`)
	  AND p.decision_type = 1 AND p.cube_value > 0 AND p.cube_owner <> -1)`, append(args, mvArgs...)...)
	if err != nil {
		return errf(db, "drop the statistics of owned-cube answers", err)
	}
	return nil
}
