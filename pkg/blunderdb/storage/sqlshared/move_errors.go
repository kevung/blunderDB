package sqlshared

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// scoredMoveCols reads what PlayScorer needs from a move and its position's
// analysis, joined as mv and a.
func scoredMoveCols(d Dialect) string {
	return `mv.id, COALESCE(mv.position_id, 0), ` + ActionLabelOrEmptyFor(d, "mv.move_type") + `,
	COALESCE(mv.checker_move, ''), ` + ActionLabelOrEmptyFor(d, "mv.cube_action") + `, a.data`
}

type moveScore struct {
	id  int64
	err *int32
}

// scoreRows scores every row of scoredMoveCols, decoding each position's
// analysis once.
func scoreRows(rows Rows) ([]moveScore, error) {
	defer rows.Close()
	scorer := PlayScorer{}
	var out []moveScore
	for rows.Next() {
		var mv domain.Move
		var data []byte
		if err := rows.Scan(&mv.ID, &mv.PositionID, &mv.MoveType, &mv.CheckerMove, &mv.CubeAction, &data); err != nil {
			return nil, err
		}
		scorer.Score(&mv, data)
		out = append(out, moveScore{id: mv.ID, err: mv.ErrorMP})
	}
	return out, rows.Err()
}

// writeScores stores each score in one transaction; a nil score is NULL.
func writeScores(ctx context.Context, db Execer, scores []moveScore) error {
	return db.Transact(ctx, func(tx Execer) error {
		for _, sc := range scores {
			var v any
			if sc.err != nil {
				v = *sc.err
			}
			if _, err := tx.Exec(ctx, `UPDATE move SET error_mp = ? WHERE id = ?`, v, sc.id); err != nil {
				return err
			}
		}
		return nil
	})
}

// ScoreMoves implements storage.MatchStore.ScoreMoves for both backends. The
// inner join on analysis leaves out, in SQL, the moves of unanalysed
// positions, so a restarted pass only decodes again the few analysed plays
// it could not score.
func ScoreMoves(ctx context.Context, db Execer, scope string, after int64, limit int) (int64, int, error) {
	if limit <= 0 {
		limit = 1
	}
	tenant, targs := db.TenantFilter("mv", scope)
	args := append(targs, after, limit)
	rows, err := db.Query(ctx, `SELECT `+scoredMoveCols(db)+`
		FROM move mv JOIN analysis a ON a.position_id = mv.position_id
		WHERE `+tenant+` AND mv.error_mp IS NULL AND mv.id > ?
		ORDER BY mv.id LIMIT ?`, args...)
	if err != nil {
		return 0, 0, errf(db, "score moves", err)
	}
	scores, err := scoreRows(rows)
	if err != nil {
		return 0, 0, errf(db, "score moves", err)
	}
	if len(scores) == 0 {
		return 0, 0, nil
	}
	scored := scores[:0:0]
	for _, sc := range scores {
		if sc.err != nil {
			scored = append(scored, sc)
		}
	}
	if err := writeScores(ctx, db, scored); err != nil {
		return 0, 0, errf(db, "score moves", err)
	}
	return scores[len(scores)-1].id, len(scored), nil
}

// RescorePositionMoves implements storage.MatchStore.RescorePositionMoves for
// both backends.
func RescorePositionMoves(ctx context.Context, db Execer, scope string, positionID int64) error {
	tenant, targs := db.TenantFilter("mv", scope)
	args := append(targs, positionID)
	rows, err := db.Query(ctx, `SELECT `+scoredMoveCols(db)+`
		FROM move mv LEFT JOIN analysis a ON a.position_id = mv.position_id
		WHERE `+tenant+` AND mv.position_id = ?`, args...)
	if err != nil {
		return errf(db, "rescore position moves", err)
	}
	scores, err := scoreRows(rows)
	if err != nil {
		return errf(db, "rescore position moves", err)
	}
	if err := writeScores(ctx, db, scores); err != nil {
		return errf(db, "rescore position moves", err)
	}
	_, err = RescorePlayedDecisionsOf(ctx, db, []int64{positionID})
	return err
}
