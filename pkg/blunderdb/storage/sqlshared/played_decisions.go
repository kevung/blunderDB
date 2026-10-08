package sqlshared

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// played_decisions.go — the error of a decision as played in its match.
//
// A Position is deduplicated across matches by its Zobrist hash, so its one
// analysis row cannot say what each match played there: its error columns
// score one play (the first the analysis or the move table names). The
// statistics read a decision's error and its counted flag from the move
// instead (statsErrExpr, countedExpr): move.decision_error_mp and
// move.is_close_cube, the same projections as the analysis columns but
// computed against the move's own checker play and cube action. They are
// written with the move (PlayedDecisionArgs) and rewritten for every move of
// a position whenever its analysis is (RescorePlayedDecisions).

// playedDecision is what the per-move columns hold: the error of the play in
// millipoints, nil when the analysis cannot score it, and the close-cube flag.
type playedDecision struct {
	errMP     *int64
	closeCube int64
}

// scorePlayedDecision projects analysis a onto one move: decisionType is its
// position's (1 cube, 0 checker), checkerMove and cubeAction what the move
// recorded. A move that recorded nothing falls back on what the analysis
// states was played, as the analysis columns do. A nil analysis scores
// nothing.
func scorePlayedDecision(a *domain.PositionAnalysis, decisionType int, checkerMove, cubeAction string) playedDecision {
	if a == nil {
		return playedDecision{}
	}
	move, cube := engine.PlayedActionsFor([]string{checkerMove}, []string{cubeAction}, a.PlayedMoves, a.PlayedCubeActions)
	c := engine.PopulateAnalysisColumns(a, move, cube, engine.LegalPlaysUnknown)
	pd := playedDecision{closeCube: c.IsCloseCube}
	switch {
	case decisionType == 1:
		v := c.CubeError
		pd.errMP = &v
	case !c.BestMoveUnscored:
		v := c.BestMoveEquityError
		pd.errMP = &v
	}
	return pd
}

func (pd playedDecision) errArg() any {
	if pd.errMP == nil {
		return nil
	}
	return *pd.errMP
}

// PlayedDecisionAnalysisSQL reads what PlayedDecisionArgs scores a new move
// by; a constant, so an import prepares it once.
const PlayedDecisionAnalysisSQL = `SELECT COALESCE(p.decision_type, 0), a.data
		FROM position p JOIN analysis a ON a.position_id = p.id WHERE p.id = ?`

// PlayedDecisionArgs returns the values of move.decision_error_mp and
// move.is_close_cube for a move about to be written at positionID with the
// given checker play and cube action, from the position's stored analysis.
// A move without position or analysis gets NULL and 0.
func PlayedDecisionArgs(ctx context.Context, db Execer, positionID int64, checkerMove, cubeAction string) (errMP any, closeCube int64, err error) {
	if positionID == 0 {
		return nil, 0, nil
	}
	rows, err := db.Query(ctx, PlayedDecisionAnalysisSQL, positionID)
	if err != nil {
		return nil, 0, errf(db, "read analysis of move", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, 0, rows.Err()
	}
	var decisionType int
	var data []byte
	if err := rows.Scan(&decisionType, &data); err != nil {
		return nil, 0, errf(db, "read analysis of move", err)
	}
	a, decErr := engine.DecodeAnalysisFromStorage(data)
	if decErr != nil {
		return nil, 0, nil
	}
	pd := scorePlayedDecision(&a, decisionType, checkerMove, cubeAction)
	return pd.errArg(), pd.closeCube, nil
}

// playedDecisionChunk bounds the positions one rescoring query names.
const playedDecisionChunk = 500

// RescorePlayedDecisions rewrites move.decision_error_mp and
// move.is_close_cube for every move played from the positions of analyses,
// each scored by its position's analysis (nil: none, the move is unscored),
// and drops the match_stats rows of the matches whose moves changed. It
// returns how many moves changed. Only changed moves are written, so a
// second run returns 0.
//
// Positions are named by id alone: their ids are unique across tenants, and
// a caller rescoring for one tenant only holds that tenant's positions.
func RescorePlayedDecisions(ctx context.Context, db Execer, analyses map[int64]*domain.PositionAnalysis) (int, error) {
	ids := make([]int64, 0, len(analyses))
	for id := range analyses {
		ids = append(ids, id)
	}
	changed := 0
	for start := 0; start < len(ids); start += playedDecisionChunk {
		n, err := rescorePlayedDecisionsChunk(ctx, db, ids[start:min(start+playedDecisionChunk, len(ids))], analyses)
		changed += n
		if err != nil {
			return changed, err
		}
	}
	return changed, nil
}

// RescoreMovesSQL reads the moves of n positions as RescorePlayedDecisions
// scores them; for one position, what every analysis write sends, its text
// is fixed, so an import prepares it once.
func RescoreMovesSQL(d Dialect, n int) string {
	return `SELECT mv.id, mv.position_id, COALESCE(p.decision_type, 0), COALESCE(mv.checker_move, ''),
			` + ActionLabelOrEmptyFor(d, "mv.cube_action") + `, mv.decision_error_mp, COALESCE(mv.is_close_cube, 0)
		FROM move mv JOIN position p ON p.id = mv.position_id
		WHERE mv.position_id IN (` + Placeholders(n) + `)`
}

func rescorePlayedDecisionsChunk(ctx context.Context, db Execer, ids []int64, analyses map[int64]*domain.PositionAnalysis) (int, error) {
	type move struct {
		id                   int64
		positionID           int64
		decisionType         int
		checkerMove, cubeAct string
		stored               playedDecision
	}
	rows, err := db.Query(ctx, RescoreMovesSQL(db, len(ids)), int64Args(ids)...)
	if err != nil {
		return 0, errf(db, "read moves to rescore", err)
	}
	var moves []move
	for rows.Next() {
		var m move
		if err := rows.Scan(&m.id, &m.positionID, &m.decisionType, &m.checkerMove, &m.cubeAct, &m.stored.errMP, &m.stored.closeCube); err != nil {
			rows.Close()
			return 0, errf(db, "read moves to rescore", err)
		}
		moves = append(moves, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, errf(db, "read moves to rescore", err)
	}

	var changedIDs []int64
	err = db.Transact(ctx, func(tx Execer) error {
		for _, m := range moves {
			pd := scorePlayedDecision(analyses[m.positionID], m.decisionType, m.checkerMove, m.cubeAct)
			if pd.closeCube == m.stored.closeCube && equalErr(pd.errMP, m.stored.errMP) {
				continue
			}
			if _, err := tx.Exec(ctx, `UPDATE move SET decision_error_mp = ?, is_close_cube = ? WHERE id = ?`,
				pd.errArg(), pd.closeCube, m.id); err != nil {
				return fmt.Errorf("rescore move %d: %w", m.id, err)
			}
			changedIDs = append(changedIDs, m.id)
		}
		if len(changedIDs) == 0 {
			return nil
		}
		// The match_stats rows summarise the columns just rewritten.
		for start := 0; start < len(changedIDs); start += playedDecisionChunk {
			part := changedIDs[start:min(start+playedDecisionChunk, len(changedIDs))]
			if _, err := tx.Exec(ctx, `DELETE FROM match_stats WHERE match_id IN
				(SELECT g.match_id FROM move mv JOIN game g ON g.id = mv.game_id WHERE mv.id IN (`+Placeholders(len(part))+`))`,
				int64Args(part)...); err != nil {
				return fmt.Errorf("invalidate match stats of rescored moves: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return 0, errf(db, "rescore moves", err)
	}
	return len(changedIDs), nil
}

func equalErr(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// RescorePlayedDecisionsOf is RescorePlayedDecisions over positionIDs, each
// scored by its stored analysis: what a write that points moves at another
// position, or rewrites an analysis without its columns, calls afterwards.
func RescorePlayedDecisionsOf(ctx context.Context, db Execer, positionIDs []int64) (int, error) {
	analyses := make(map[int64]*domain.PositionAnalysis, len(positionIDs))
	for _, id := range positionIDs {
		analyses[id] = nil
	}
	for start := 0; start < len(positionIDs); start += playedDecisionChunk {
		part := positionIDs[start:min(start+playedDecisionChunk, len(positionIDs))]
		rows, err := db.Query(ctx, `SELECT position_id, data FROM analysis WHERE position_id IN (`+Placeholders(len(part))+`)`, int64Args(part)...)
		if err != nil {
			return 0, errf(db, "read analyses to rescore moves", err)
		}
		for rows.Next() {
			var id int64
			var data []byte
			if err := rows.Scan(&id, &data); err != nil {
				rows.Close()
				return 0, errf(db, "read analyses to rescore moves", err)
			}
			if a, err := engine.DecodeAnalysisFromStorage(data); err == nil {
				analyses[id] = &a
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return 0, errf(db, "read analyses to rescore moves", err)
		}
	}
	return RescorePlayedDecisions(ctx, db, analyses)
}
