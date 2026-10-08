package gammonnet

import (
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// A take/pass decision is stored on the position after the double: the
// answerer on roll, the cube already turned. Evaluated as it stands, that
// position is the answerer's own REDOUBLE decision, not the take. The take is
// the second half of the doubler's cube decision, so it is evaluated from the
// doubler's side before the double, the way every importer stores it (one
// cube analysis, its equities the doubler's, at the cube before the double).

// IsResponsePosition reports whether pos can only be a take/pass decision: a
// turned cube held by no one, the importers' convention (ingest/xg.go). A
// transcribed or duelled answer gives the answerer the cube instead, so only
// its move records tell it apart from a redouble.
func IsResponsePosition(pos *domain.Position) bool {
	return pos.DecisionType == domain.CubeAction && pos.Cube.Value > 0 && pos.Cube.Owner == domain.None
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

// EvaluateResponseWithMET evaluates the take/pass decision stored at pos (see
// DoublerPosition): the cube analysis of the doubler's decision, its equities
// left on the doubler's side as engine.CubeActionError reads them for a take
// or a pass, its chances turned to the answerer on roll at pos. PreRoll and
// CubeAction stay the doubler's.
func EvaluateResponseWithMET(searcher *Searcher, pos domain.Position, met *engine.MET, ply, pruneK int) (EvalResult, error) {
	res, err := EvaluatePositionWithMET(searcher, DoublerPosition(pos), met, ply, pruneK, 0)
	if err != nil || res.Cube == nil {
		return res, err
	}
	c := res.Cube
	c.PlayerWinChances, c.OpponentWinChances = c.OpponentWinChances, c.PlayerWinChances
	c.PlayerGammonChances, c.OpponentGammonChances = c.OpponentGammonChances, c.PlayerGammonChances
	c.PlayerBackgammonChances, c.OpponentBackgammonChances = c.OpponentBackgammonChances, c.PlayerBackgammonChances
	return res, nil
}
