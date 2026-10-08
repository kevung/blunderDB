package gammonnet

import (
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// A take/pass decision is stored on the position after the double: the
// answerer on roll, the cube turned and held by no one (ingest/xg.go,
// transcript's answeredCube). The take is the second half of the doubler's
// cube decision, so it is evaluated from the doubler's side before the
// double, the way every importer stores it: one cube analysis, its equities
// the doubler's, at the cube before the double.

// IsResponsePosition reports whether pos is a take/pass decision
// (domain.IsResponsePosition).
func IsResponsePosition(pos *domain.Position) bool {
	return domain.IsResponsePosition(pos)
}

// DoublerPosition rebuilds, from a take/pass position, the doubler's position
// before the double: the doubler on roll, the cube one level down, owned by
// the doubler unless it was centred.
func DoublerPosition(pos domain.Position) domain.Position {
	pos.PlayerOnRoll = 1 - pos.PlayerOnRoll
	pos.DecisionType = domain.CubeAction
	pos.Dice = [2]int{}
	if pos.Cube.Value > 0 {
		pos.Cube.Value--
	}
	pos.Cube.Owner = domain.None
	if pos.Cube.Value > 0 {
		pos.Cube.Owner = pos.PlayerOnRoll
	}
	return pos
}

// evaluateResponse evaluates the take/pass decision stored at pos: the cube
// analysis of the doubler's decision, its equities and CubeAction left on the
// doubler's side as engine.CubeActionError reads them for a take or a pass,
// its chances turned to the answerer on roll at pos. PreRoll is turned too,
// its cubeless equity still in units of the cube before the double.
func evaluateResponse(searcher *Searcher, pos domain.Position, met *engine.MET, ply, pruneK int) (EvalResult, error) {
	res, err := EvaluatePositionWithMET(searcher, DoublerPosition(pos), met, ply, pruneK, 0)
	if err != nil {
		return res, err
	}
	if c := res.Cube; c != nil {
		c.PlayerWinChances, c.OpponentWinChances = c.OpponentWinChances, c.PlayerWinChances
		c.PlayerGammonChances, c.OpponentGammonChances = c.OpponentGammonChances, c.PlayerGammonChances
		c.PlayerBackgammonChances, c.OpponentBackgammonChances = c.OpponentBackgammonChances, c.PlayerBackgammonChances
	}
	if f := res.PreRoll; f != nil {
		f.PlayerWinChance, f.OpponentWinChance = f.OpponentWinChance, f.PlayerWinChance
		f.PlayerGammonChance, f.OpponentGammonChance = f.OpponentGammonChance, f.PlayerGammonChance
		f.PlayerBackgammonChance, f.OpponentBackgammonChance = f.OpponentBackgammonChance, f.PlayerBackgammonChance
		f.CubelessEquity = -f.CubelessEquity
	}
	return res, nil
}
