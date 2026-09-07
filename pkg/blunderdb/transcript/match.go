package transcript

import (
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Parts is what a Transcription becomes once it leaves this package: a match, its
// games, its moves, and the Position each move was played from. Nothing here is
// stored yet — it is the caller, on the database side, that turns it into an ingest
// graph and writes it, and the .mat renderer that takes the first three as they are.
//
// Moves and Positions are keyed by game id and are index-aligned: Positions[g][i] is
// the position of Moves[g][i].
type Parts struct {
	Match     *domain.Match
	Games     []*domain.Game
	Moves     map[int64][]*domain.Move
	Positions map[int64][]domain.Position
}

// MatchParts returns the triplet ingest.RenderMAT takes, and that a save writes: the
// match, its games in order, and each game's moves keyed by game id.
//
// A game is one Game with its initial score, its winner and its points; every checker,
// dance, double, take and pass Action is one Move. A resignation is none of them: it
// leaves winner and points on the Game and adds no row to `move` (ADR-0045 §6), and an
// opening produces nothing at all.
//
// The Position of each Move is not in the triplet — domain.Move carries an id, not a
// position — so a caller that needs them takes [Build] instead, or reads them off the
// Replay: ActionInfo.GameIndex and ActionInfo.MoveNumber name the Move an Action
// became, and ActionInfo.Before is its Position.
func MatchParts(doc Document) (*domain.Match, []*domain.Game, map[int64][]*domain.Move) {
	p := Build(doc)
	return p.Match, p.Games, p.Moves
}

// Build derives everything a save and an export need from the document, in one Replay.
func Build(doc Document) Parts {
	ann := Replay(doc, 0)
	h := doc.Header

	m := &domain.Match{
		Player1Name:  h.Player1,
		Player2Name:  h.Player2,
		Event:        h.Event,
		Location:     h.Location,
		Round:        h.Round,
		MatchLength:  int32(max(h.MatchLength, 0)),
		MatchDate:    h.Date,
		GameCount:    len(ann.Games),
		TournamentID: h.TournamentID,
	}
	if h.MatchID != nil {
		m.ID = *h.MatchID
	}

	games := make([]*domain.Game, 0, len(ann.Games))
	for i, g := range ann.Games {
		games = append(games, &domain.Game{
			ID:           int64(i + 1),
			MatchID:      m.ID,
			GameNumber:   int32(g.Number),
			InitialScore: [2]int32{int32(g.InitialScore[0]), int32(g.InitialScore[1])},
			Winner:       int32(g.Winner),
			PointsWon:    int32(g.PointsWon),
		})
	}

	moves := map[int64][]*domain.Move{}
	positions := map[int64][]domain.Position{}
	for _, info := range ann.Actions {
		if info.MoveNumber < 0 || info.GameIndex < 0 || info.GameIndex >= len(games) {
			continue
		}
		gameID := games[info.GameIndex].ID
		mv := &domain.Move{
			GameID:     gameID,
			MoveNumber: info.MoveNumber,
			Player:     sideToXG(info.Side),
		}
		switch info.Kind {
		case KindChecker, KindDance:
			mv.MoveType = "checker"
			mv.Dice = [2]int32{int32(info.Before.Dice[0]), int32(info.Before.Dice[1])}
			mv.CheckerMove = info.Notation
		case KindDouble:
			mv.MoveType, mv.CubeAction = "cube", "Double"
		case KindTake:
			mv.MoveType, mv.CubeAction = "cube", "Take"
		case KindPass:
			mv.MoveType, mv.CubeAction = "cube", "Pass"
		default:
			continue
		}
		moves[gameID] = append(moves[gameID], mv)
		positions[gameID] = append(positions[gameID], info.Before)
	}
	for _, g := range games {
		g.MoveCount = len(moves[g.ID])
	}

	return Parts{Match: m, Games: games, Moves: moves, Positions: positions}
}

// sideToXG converts a Transcription's side (0 = player 1) into the encoding
// domain.Move carries, which is XG's: +1 for player 1, -1 for player 2.
func sideToXG(side int) int32 {
	if side == domain.Black {
		return 1
	}
	return -1
}
