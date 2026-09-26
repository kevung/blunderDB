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
		`SELECT mv.id, mv.position_id, COALESCE(mv.move_type, ''),
			COALESCE(mv.checker_move, ''), COALESCE(mv.cube_action, ''), a.data
		 FROM move mv
		 JOIN game g ON g.id = mv.game_id
		 JOIN position p ON p.id = mv.position_id
		 JOIN analysis a ON a.position_id = p.id
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
