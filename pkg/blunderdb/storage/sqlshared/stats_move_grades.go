package sqlshared

import (
	"context"
	"math"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// MatchMoveGrades scores every Move of a match by its own play: a
// checker Move is looked up among its Position's analysed candidates, a cube
// Move goes through engine.CubeActionError — the two rules the search's `E`
// filter scores a play by. The grade is drawn at the library's thresholds
// (ADR-0046), inclusive like everywhere else: a cost of exactly the threshold
// is on the wrong side of the line.
//
// The denormalised error column is deliberately not read: it scores the FIRST
// play of a Position, and a Position deduplicated within a match (an opening
// reached in two games, played two ways) would give both Moves the first
// one's grade. Each blob is decoded once per Position, however many
// Moves reach it.
func (s *StatsStore) MatchMoveGrades(ctx context.Context, scope string, matchID int64) ([]storage.MoveGrade, error) {
	settings, err := librarySettings(ctx, s.DB, scope)
	if err != nil {
		return nil, errf(s.DB, "MatchMoveGrades settings", err)
	}
	tenant, args := s.DB.TenantFilter("p", scope)
	rows, err := s.DB.Query(ctx,
		`SELECT mv.id, mv.position_id, `+ActionLabelOrEmptyFor(s.DB, "mv.move_type")+`,
			COALESCE(mv.checker_move, ''), `+ActionLabelOrEmptyFor(s.DB, "mv.cube_action")+`, a.data
		 FROM move mv
		 JOIN game g ON g.id = mv.game_id`+andSameTenant(s.DB, "g", "mv")+`
		 JOIN position p ON p.id = mv.position_id`+andSameTenant(s.DB, "p", "mv")+`
		 JOIN analysis a ON a.position_id = p.id`+andSameTenant(s.DB, "a", "p")+`
		 WHERE `+tenant+` AND g.match_id = ?
		 ORDER BY g.game_number, mv.move_number, mv.id`,
		append(args, matchID)...)
	if err != nil {
		return nil, errf(s.DB, "MatchMoveGrades query", err)
	}
	defer rows.Close()

	decoded := make(map[int64]*domain.PositionAnalysis)
	var grades []storage.MoveGrade
	for rows.Next() {
		var moveID, positionID int64
		var moveType, checkerMove, cubeAction string
		var data []byte
		if err := rows.Scan(&moveID, &positionID, &moveType, &checkerMove, &cubeAction, &data); err != nil {
			return nil, errf(s.DB, "MatchMoveGrades scan", err)
		}
		analysis, seen := decoded[positionID]
		if !seen {
			if a, err := engine.DecodeAnalysisFromStorage(data); err == nil {
				analysis = &a
			}
			decoded[positionID] = analysis // nil too: an undecodable blob is not retried
		}
		if analysis == nil {
			continue
		}
		cost, ok := playError(analysis, moveType, checkerMove, cubeAction)
		if !ok {
			continue
		}
		errMP := int(math.Round(cost * 1000))
		grade := ""
		switch {
		case errMP >= settings.BlunderThresholdMP:
			grade = storage.MoveGradeBlunder
		case errMP >= settings.ErrorThresholdMP:
			grade = storage.MoveGradeError
		}
		grades = append(grades, storage.MoveGrade{MoveID: moveID, ErrorMP: errMP, Grade: grade})
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "MatchMoveGrades rows", err)
	}
	return grades, nil
}

// playError is the cost of one recorded play as its Position's analysis
// scores it, in equity units, always non-negative. The Move's type says which
// half of the analysis applies: a cube Move is scored by the cube analysis
// even when the blob also carries candidates, and the other way round.
func playError(analysis *domain.PositionAnalysis, moveType, checkerMove, cubeAction string) (float64, bool) {
	if moveType == "cube" {
		if cubeAction == "" {
			return 0, false
		}
		e, ok := engine.CubeActionError(analysis.DoublingCubeAnalysis, cubeAction)
		return math.Abs(e), ok
	}
	if checkerMove == "" || analysis.CheckerAnalysis == nil {
		return 0, false
	}
	return checkerPlayError(analysis.CheckerAnalysis, checkerMove)
}

// checkerPlayError looks a played checker move up among the analysed
// candidates: the best one costs 0, any other its recorded equity error. A
// move absent from the list is not scored. Shared with the search's move-error
// filter (player1MaxMoveError), so the Transcript and `E>x` cannot disagree
// on what a play cost.
func checkerPlayError(ca *domain.CheckerAnalysis, played string) (float64, bool) {
	normPlayed := engine.NormalizeMove(played)
	for i, m := range ca.Moves {
		if !strings.EqualFold(engine.NormalizeMove(m.Move), normPlayed) {
			continue
		}
		if i > 0 && m.EquityError != nil {
			return math.Abs(*m.EquityError), true
		}
		return 0, true
	}
	return 0, false
}

// bestPlay names what the analysis ranks first for the decision a Move
// records, with the same half-of-the-analysis rule as playError. The doubling
// decision follows the engine's own ruling (bestCubeAction), read by
// BestCubeVerdict; the answer to a double follows the equities, as
// CubeActionError scores it. The equities stand in for a missing or unknown
// label; an analysis carrying neither names no best action.
func bestPlay(analysis *domain.PositionAnalysis, moveType, cubeAction string) (checker, cube string) {
	if moveType != "cube" {
		if ca := analysis.CheckerAnalysis; ca != nil && len(ca.Moves) > 0 {
			checker = ca.Moves[0].Move
		}
		return checker, ""
	}
	dca := analysis.DoublingCubeAnalysis
	if dca == nil {
		return "", ""
	}
	verdict, labelled := engine.BestCubeVerdict(dca.BestCubeAction)
	switch engine.CanonicalCubeAction(cubeAction) {
	case engine.CubeTake, engine.CubePass:
		// The analysis is the doubler's: the responder's best answer is the
		// one leaving the doubler the lower equity.
		pass := dca.CubefulDoublePassEquity < dca.CubefulDoubleTakeEquity
		if dca.CubefulDoublePassEquity == 0 && dca.CubefulDoubleTakeEquity == 0 {
			if !labelled {
				return "", ""
			}
			pass = verdict.ShouldPass
		}
		if pass {
			return "", "Pass"
		}
		return "", "Take"
	case engine.CubeDouble, engine.CubeNoDouble:
		double := verdict.ShouldDouble
		if !labelled {
			if dca.CubefulNoDoubleEquity == 0 && dca.CubefulDoubleTakeEquity == 0 && dca.CubefulDoublePassEquity == 0 {
				return "", ""
			}
			double = math.Min(dca.CubefulDoubleTakeEquity, dca.CubefulDoublePassEquity) > dca.CubefulNoDoubleEquity
		}
		if double {
			return "", "Double"
		}
		return "", "No Double"
	}
	return "", ""
}

// PlayScorer fills Move.ErrorMP, BestMove and BestCubeAction from the
// analysis blob read beside each Move, decoding a Position's blob once however
// many Moves reach it. A nil blob (no analysis) or an undecodable one leaves
// the Move unscored.
type PlayScorer map[int64]*domain.PositionAnalysis

// Score sets mv.ErrorMP and mv's best play from data, the stored analysis of
// mv's Position.
func (p PlayScorer) Score(mv *domain.Move, data []byte) {
	if len(data) == 0 || mv.PositionID == 0 {
		return
	}
	analysis, seen := p[mv.PositionID]
	if !seen {
		if a, err := engine.DecodeAnalysisFromStorage(data); err == nil {
			analysis = &a
		}
		p[mv.PositionID] = analysis
	}
	if analysis == nil {
		return
	}
	mv.BestMove, mv.BestCubeAction = bestPlay(analysis, mv.MoveType, mv.CubeAction)
	cost, ok := playError(analysis, mv.MoveType, mv.CheckerMove, mv.CubeAction)
	if !ok {
		return
	}
	errMP := int32(math.Round(cost * 1000))
	mv.ErrorMP = &errMP
}
