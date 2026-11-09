package sqlshared

import (
	"context"
	"fmt"
	"strings"

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
func scorePlayedDecision(a *domain.PositionAnalysis, pos *domain.Position, decisionType int, checkerMove, cubeAction string) playedDecision {
	if a == nil {
		return playedDecision{}
	}
	move, cube := engine.PlayedActionsFor([]string{checkerMove}, []string{cubeAction}, a.PlayedMoves, a.PlayedCubeActions)
	c := engine.PlayedDecisionColumns(a, move, cube, pos)
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
// by; its text is fixed per dialect, so an import prepares it once. Its
// arguments are the position id, then the tenant where the dialect has one
// (andTenant): position is keyed by (tenant_id, id).
func PlayedDecisionAnalysisSQL(d Dialect) string {
	tenant, _ := andTenant(d, "p", "")
	return `SELECT COALESCE(p.decision_type, 0), a.data, ` + PlayPositionColumns + `
		FROM position p JOIN analysis a ON a.position_id = p.id` + andSameTenant(d, "a", "p") + ` WHERE p.id = ?` + tenant
}

// andTenant is a conjunct confining alias to the scope's tenant, and its
// argument, on a dialect whose rows carry one; nothing on SQLite, whose
// TenantFilter is a constant TRUE that would change a prepared text.
func andTenant(d Dialect, alias, scope string) (string, []any) {
	cols, args := d.TenantColumns(scope)
	if len(cols) == 0 {
		return "", nil
	}
	return " AND " + alias + "." + cols[0] + " = ?", args
}

// PlayedDecisionArgs returns the values of move.decision_error_mp and
// move.is_close_cube for a move about to be written at positionID with the
// given checker play and cube action, from the position's stored analysis.
// A move without position or analysis gets NULL and 0.
func PlayedDecisionArgs(ctx context.Context, db Execer, scope string, positionID int64, checkerMove, cubeAction string) (errMP any, closeCube int64, err error) {
	if positionID == 0 {
		return nil, 0, nil
	}
	_, tenantArgs := andTenant(db, "p", scope)
	rows, err := db.Query(ctx, PlayedDecisionAnalysisSQL(db), append([]any{positionID}, tenantArgs...)...)
	if err != nil {
		return nil, 0, errf(db, "read analysis of move", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, 0, rows.Err()
	}
	basis, err := ScanPlayedDecisionBasis(rows.Scan)
	if err != nil {
		return nil, 0, errf(db, "read analysis of move", err)
	}
	errMP, closeCube = basis.Args(checkerMove, cubeAction)
	return errMP, closeCube, nil
}

// PlayedDecisionBasis is what a move at one position is scored by: the row
// PlayedDecisionAnalysisSQL reads, decoded once so that every move played
// from the position can be scored without reading it again.
type PlayedDecisionBasis struct {
	decisionType int
	analysis     *domain.PositionAnalysis
	position     *domain.Position
}

// NewPlayedDecisionBasis is the basis of a position whose analysis the caller
// holds decoded, as PlayedDecisionAnalysisSQL would read it back: a writer
// that has just stored a can score the position's moves without decoding it
// again. pos is PlayPosition of the stored row.
func NewPlayedDecisionBasis(decisionType int, a *domain.PositionAnalysis, pos *domain.Position) *PlayedDecisionBasis {
	return &PlayedDecisionBasis{decisionType: decisionType, analysis: a, position: pos}
}

// ScanPlayedDecisionBasis decodes one row of PlayedDecisionAnalysisSQL. An
// analysis that cannot be decoded gives a nil basis, which scores NULL and 0
// like a position without analysis.
func ScanPlayedDecisionBasis(scan func(dest ...any) error) (*PlayedDecisionBasis, error) {
	var decisionType int
	var data, state []byte
	var por, d1, d2, dt *int64
	if err := scan(&decisionType, &data, &state, &por, &d1, &d2, &dt); err != nil {
		return nil, err
	}
	a, decErr := engine.DecodeAnalysisFromStorage(data)
	if decErr != nil {
		return nil, nil
	}
	return &PlayedDecisionBasis{decisionType: decisionType, analysis: &a, position: PlayPosition(state, por, d1, d2, dt)}, nil
}

// Args returns move.decision_error_mp and move.is_close_cube for a move with
// the given play; a nil basis (no analysis) gives NULL and 0.
func (b *PlayedDecisionBasis) Args(checkerMove, cubeAction string) (errMP any, closeCube int64) {
	if b == nil {
		return nil, 0
	}
	pd := scorePlayedDecision(b.analysis, b.position, b.decisionType, checkerMove, cubeAction)
	return pd.errArg(), pd.closeCube
}

// playedDecisionChunk bounds the positions one rescoring query names.
const playedDecisionChunk = 500

// playedDecisionUpdateBatch is how many moves one UPDATE rewrites.
const playedDecisionUpdateBatch = 200

// RescorePlayedDecisions rewrites move.decision_error_mp and
// move.is_close_cube for every move played from the positions of analyses,
// each scored by its position's analysis (nil: none, the move is unscored),
// and drops the match_stats rows of the matches whose moves changed. It
// returns how many moves changed. Only changed moves are written, so a
// second run returns 0.
//
// The positions are scope's: the keys of move and match_stats lead with the
// tenant, so each write names it.
func RescorePlayedDecisions(ctx context.Context, db Execer, scope string, analyses map[int64]*domain.PositionAnalysis) (int, error) {
	ids := make([]int64, 0, len(analyses))
	for id := range analyses {
		ids = append(ids, id)
	}
	changed := 0
	for start := 0; start < len(ids); start += playedDecisionChunk {
		n, err := rescorePlayedDecisionsChunk(ctx, db, scope, ids[start:min(start+playedDecisionChunk, len(ids))], analyses)
		changed += n
		if err != nil {
			return changed, err
		}
	}
	return changed, nil
}

// RescoreMovesSQL reads the moves of n positions as RescorePlayedDecisions
// scores them; for one position, what every analysis write sends, its text
// is fixed, so an import prepares it once. Its first argument is the tenant
// (TenantFilter's, none on SQLite): the text of the predicate does not
// depend on which.
func RescoreMovesSQL(d Dialect, n int) string {
	tenant, _ := d.TenantFilter("mv", "")
	return `SELECT mv.id, mv.position_id, COALESCE(p.decision_type, 0), COALESCE(mv.checker_move, ''),
			` + ActionLabelOrEmptyFor(d, "mv.cube_action") + `, mv.decision_error_mp, COALESCE(mv.is_close_cube, 0),
			` + PlayPositionColumns + `
		FROM move mv JOIN position p ON p.id = mv.position_id` + andSameTenant(d, "p", "mv") + `
		WHERE ` + tenant + ` AND mv.position_id IN (` + Placeholders(n) + `)`
}

func rescorePlayedDecisionsChunk(ctx context.Context, db Execer, scope string, ids []int64, analyses map[int64]*domain.PositionAnalysis) (int, error) {
	type move struct {
		id                   int64
		positionID           int64
		decisionType         int
		checkerMove, cubeAct string
		stored               playedDecision
		pos                  *domain.Position
	}
	_, targs := db.TenantFilter("mv", scope)
	rows, err := db.Query(ctx, RescoreMovesSQL(db, len(ids)), append(append([]any{}, targs...), int64Args(ids)...)...)
	if err != nil {
		return 0, errf(db, "read moves to rescore", err)
	}
	var moves []move
	for rows.Next() {
		var m move
		var state []byte
		var por, d1, d2, dt *int64
		if err := rows.Scan(&m.id, &m.positionID, &m.decisionType, &m.checkerMove, &m.cubeAct, &m.stored.errMP, &m.stored.closeCube, &state, &por, &d1, &d2, &dt); err != nil {
			rows.Close()
			return 0, errf(db, "read moves to rescore", err)
		}
		m.pos = PlayPosition(state, por, d1, d2, dt)
		moves = append(moves, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, errf(db, "read moves to rescore", err)
	}

	var changedIDs []int64
	err = db.Transact(ctx, func(tx Execer) error {
		type update struct {
			id int64
			pd playedDecision
		}
		var updates []update
		moveTenant, moveArgs := tx.TenantFilter("move", scope)
		mvTenant, mvArgs := tx.TenantFilter("mv", scope)
		statsTenant, statsArgs := tx.TenantFilter("", scope)
		for _, m := range moves {
			pd := scorePlayedDecision(analyses[m.positionID], m.pos, m.decisionType, m.checkerMove, m.cubeAct)
			if pd.closeCube == m.stored.closeCube && equalErr(pd.errMP, m.stored.errMP) {
				continue
			}
			updates = append(updates, update{m.id, pd})
			changedIDs = append(changedIDs, m.id)
		}
		// One statement per batch of moves, not one per move.
		for start := 0; start < len(updates); start += playedDecisionUpdateBatch {
			part := updates[start:min(start+playedDecisionUpdateBatch, len(updates))]
			args := make([]any, 0, 3*len(part))
			var sb strings.Builder
			sb.WriteString(`UPDATE move SET decision_error_mp = v.e, is_close_cube = v.c FROM (`)
			for i, u := range part {
				if i > 0 {
					sb.WriteString(" UNION ALL ")
				}
				sb.WriteString(`SELECT CAST(? AS BIGINT) AS id, CAST(? AS BIGINT) AS e, CAST(? AS BIGINT) AS c`)
				args = append(args, u.id, u.pd.errArg(), u.pd.closeCube)
			}
			sb.WriteString(`) AS v WHERE ` + moveTenant + ` AND move.id = v.id`)
			if _, err := tx.Exec(ctx, sb.String(), append(args, moveArgs...)...); err != nil {
				return fmt.Errorf("rescore moves: %w", err)
			}
		}
		if len(changedIDs) == 0 {
			return nil
		}
		// The match_stats rows summarise the columns just rewritten.
		for start := 0; start < len(changedIDs); start += playedDecisionChunk {
			part := changedIDs[start:min(start+playedDecisionChunk, len(changedIDs))]
			args := append(append(append([]any{}, statsArgs...), mvArgs...), int64Args(part)...)
			if _, err := tx.Exec(ctx, `DELETE FROM match_stats WHERE `+statsTenant+` AND match_id IN
				(SELECT g.match_id FROM move mv JOIN game g ON g.id = mv.game_id`+andSameTenant(tx, "g", "mv")+`
				 WHERE `+mvTenant+` AND mv.id IN (`+Placeholders(len(part))+`))`, args...); err != nil {
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
func RescorePlayedDecisionsOf(ctx context.Context, db Execer, scope string, positionIDs []int64) (int, error) {
	tenant, targs := db.TenantFilter("", scope)
	analyses := make(map[int64]*domain.PositionAnalysis, len(positionIDs))
	for _, id := range positionIDs {
		analyses[id] = nil
	}
	for start := 0; start < len(positionIDs); start += playedDecisionChunk {
		part := positionIDs[start:min(start+playedDecisionChunk, len(positionIDs))]
		rows, err := db.Query(ctx, `SELECT position_id, data FROM analysis WHERE `+tenant+` AND position_id IN (`+Placeholders(len(part))+`)`,
			append(append([]any{}, targs...), int64Args(part)...)...)
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
	return RescorePlayedDecisions(ctx, db, scope, analyses)
}

// RescoreAllPlayedDecisions runs RescorePlayedDecisions over every analysed
// position the handle sees, a page of analyses at a time, decoding each
// once: the pass that scores a library older than the per-move columns. An
// undecodable analysis leaves its moves as they are.
func RescoreAllPlayedDecisions(ctx context.Context, db Execer, scope string) (int, error) {
	changed := 0
	var last int64
	tenant, targs := db.TenantFilter("", scope)
	for {
		rows, err := db.Query(ctx, `SELECT id, position_id, data FROM analysis WHERE `+tenant+` AND id > ? ORDER BY id LIMIT ?`,
			append(append([]any{}, targs...), last, playedDecisionChunk)...)
		if err != nil {
			return changed, errf(db, "read analyses to rescore moves", err)
		}
		analyses := make(map[int64]*domain.PositionAnalysis, playedDecisionChunk)
		n := 0
		for rows.Next() {
			var id, posID int64
			var data []byte
			if err := rows.Scan(&id, &posID, &data); err != nil {
				rows.Close()
				return changed, errf(db, "read analyses to rescore moves", err)
			}
			last, n = id, n+1
			if a, err := engine.DecodeAnalysisFromStorage(data); err == nil {
				analyses[posID] = &a
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return changed, errf(db, "read analyses to rescore moves", err)
		}
		if n == 0 {
			return changed, nil
		}
		k, err := RescorePlayedDecisions(ctx, db, scope, analyses)
		changed += k
		if err != nil {
			return changed, err
		}
	}
}
