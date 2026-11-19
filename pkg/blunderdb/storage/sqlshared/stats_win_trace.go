package sqlshared

import (
	"context"
	"errors"
	"math"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// MatchWinTrace reads each Move's losses through MatchDecisionLosses, so the
// trace and the per-decision losses cannot disagree, and adds the equity of
// the decision's best option, converted at the position's score and cube by
// the conversion the losses use (engine.EquityToMWC beside
// ConvertEMGLossToMWCLoss).
//
// The stored position is the side of the seat that played the Move: its
// first score is that seat's. A cube analysis is the doubler's equity at the
// cube before the double. A response on the turned cube is the responder's
// row, so it reads the analysis from the doubler's side and turns it over; a
// take the .xg fallback left on the doubler's own row (ADR-0083, rule 2) is
// the doubler's side already and reads it as stored.
func (s *StatsStore) MatchWinTrace(ctx context.Context, scope string, matchID int64) (storage.MatchWinTrace, error) {
	trace := storage.MatchWinTrace{Decisions: []storage.WinTracePoint{}}
	tenant, args := s.DB.TenantFilter("m", scope)
	var matchLength int
	err := s.DB.QueryRow(ctx, `SELECT COALESCE(m.match_length, 0) FROM match m WHERE `+tenant+` AND m.id = ?`,
		append(args, matchID)...).Scan(&matchLength)
	if errors.Is(err, ErrNoRows) {
		return trace, nil
	}
	if err != nil {
		return trace, errf(s.DB, "MatchWinTrace match", err)
	}
	trace.Money = engine.IsMoneyLength(matchLength)

	losses, err := s.MatchDecisionLosses(ctx, scope, matchID)
	if err != nil {
		return trace, err
	}
	equities, err := s.decisionEquities(ctx, scope, matchID)
	if err != nil {
		return trace, err
	}
	for _, d := range losses {
		pt := storage.WinTracePoint{
			MoveID: d.MoveID, GameNumber: d.GameNumber, MoveNumber: d.MoveNumber,
			Player: d.Player, DecisionType: d.DecisionType, MWCLoss: d.MWCLoss, Error: d.Error,
		}
		if e, ok := equities[d.MoveID]; ok {
			pt.Analysed = true
			if !trace.Money && !math.IsNaN(e.before) {
				before := e.before
				if d.Player == 1 {
					before = 1 - before
				}
				pt.MWCBefore = &before
				// A decision outside the counted set (a forced play, a no
				// double outside the window) or an unscored play has no
				// loss: it leaves the chances where they were.
				after := before
				if d.MWCLoss != nil {
					if d.Player == 1 {
						after = before + *d.MWCLoss
					} else {
						after = before - *d.MWCLoss
					}
				}
				pt.MWCAfter = &after
			}
		}
		trace.Decisions = append(trace.Decisions, pt)
	}
	return trace, nil
}

// decisionEquity is a Move's best option in the match winning chances of
// the seat that played the Move; NaN where the score cannot be priced (money).
type decisionEquity struct {
	before float64
}

// decisionEquities maps every Move of the match whose analysis gives its best
// option to that option's chances for the decider. A Move without analysis, or
// whose analysis lacks its decision's part, is absent.
func (s *StatsStore) decisionEquities(ctx context.Context, scope string, matchID int64) (map[int64]decisionEquity, error) {
	tenant, args := s.DB.TenantFilter("mv", scope)
	rows, err := s.DB.Query(ctx,
		`SELECT mv.id, `+ActionLabelOrEmptyFor(s.DB, "mv.move_type")+`, `+ActionLabelOrEmptyFor(s.DB, "mv.cube_action")+`,
			p.id, a.data, COALESCE(p.score_1, 0), COALESCE(p.score_2, 0),
			COALESCE(p.cube_value, 0), COALESCE(p.cube_owner, -1), `+cubeMultiplierExpr+`, COALESCE(p.match_length, m.match_length, 0)
		 FROM move mv
		 JOIN game g ON g.id = mv.game_id`+andSameTenant(s.DB, "g", "mv")+`
		 JOIN match m ON m.id = g.match_id`+andSameTenant(s.DB, "m", "g")+`
		 JOIN position p ON p.id = mv.position_id`+andSameTenant(s.DB, "p", "mv")+`
		 JOIN analysis a ON a.position_id = p.id`+andSameTenant(s.DB, "a", "p")+`
		 WHERE `+tenant+` AND g.match_id = ?`,
		append(args, matchID)...)
	if err != nil {
		return nil, errf(s.DB, "MatchWinTrace query", err)
	}
	defer rows.Close()

	decoded := make(map[int64]*domain.PositionAnalysis)
	out := make(map[int64]decisionEquity)
	for rows.Next() {
		var moveID, positionID int64
		var moveType, cubeAction string
		var data []byte
		var away0, away1, storedCube, cubeOwner, cubeValue, matchLength int
		if err := rows.Scan(&moveID, &moveType, &cubeAction, &positionID, &data, &away0, &away1, &storedCube, &cubeOwner, &cubeValue, &matchLength); err != nil {
			return nil, errf(s.DB, "MatchWinTrace scan", err)
		}
		analysis, seen := decoded[positionID]
		if !seen {
			if a, err := engine.DecodeAnalysisFromStorage(data); err == nil {
				analysis = &a
			}
			decoded[positionID] = analysis
		}
		// The condition cubeMultiplierExpr steps the cube down on: a
		// response on the turned cube held by no one, stored from the
		// responder's side. The .xg fallback leaves a take on the doubler's
		// own row, an owned cube, stored from the doubler's side.
		turned := storedCube > 0 && cubeOwner == -1
		before, ok := bestOptionMWC(analysis, moveType == "cube", cubeAction, turned, away0, away1, cubeValue, matchLength)
		if !ok {
			continue
		}
		out[moveID] = decisionEquity{before: before}
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "MatchWinTrace rows", err)
	}
	return out, nil
}

// bestOptionMWC is the decider's match winning chances with the decision
// taken at its best, from the analysis its loss is scored by (ColumnSource).
// ok is false when that analysis lacks the decision's part; the chances are
// NaN when the score cannot be priced. away0 is the away score of the side
// the position is stored from; turned marks a response stored from the
// responder's side.
func bestOptionMWC(analysis *domain.PositionAnalysis, cube bool, cubeAction string, turned bool, away0, away1, cubeValue, matchLength int) (float64, bool) {
	analysis = analysis.ColumnSource()
	if analysis == nil {
		return 0, false
	}
	score0 := matchLength - domain.PointsAway(away0)
	score1 := matchLength - domain.PointsAway(away1)
	if !cube {
		ca := analysis.CheckerAnalysis
		if ca == nil || len(ca.Moves) == 0 {
			return 0, false
		}
		return engine.EquityToMWC(ca.Moves[0].Equity, score0, score1, cubeValue, matchLength), true
	}
	dca := analysis.DoublingCubeAnalysis
	if dca == nil {
		return 0, false
	}
	// The doubler's equity once the cube is offered: the responder answers
	// at its best.
	offered := math.Min(dca.CubefulDoubleTakeEquity, dca.CubefulDoublePassEquity)
	switch engine.CanonicalCubeAction(cubeAction) {
	case engine.CubeTake, engine.CubePass:
		if turned {
			return 1 - engine.EquityToMWC(offered, score1, score0, cubeValue, matchLength), true
		}
	}
	return engine.EquityToMWC(math.Max(dca.CubefulNoDoubleEquity, offered), score0, score1, cubeValue, matchLength), true
}
