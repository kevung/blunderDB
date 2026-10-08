package gammonnet

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The reply to a double, stored the importers' way (the turned cube held by
// no one, the answerer on roll), is the doubler's cube decision seen from the
// other side: same equities, chances turned to the answerer.
func TestEvaluateResponseIsTheDoublersDecisionFromTheOtherSide(t *testing.T) {
	doubler := cubeMatrixPosition(t)
	doubler.DecisionType = domain.CubeAction
	doubler.Dice = [2]int{}

	answer := doubler
	answer.PlayerOnRoll = 1 - doubler.PlayerOnRoll
	answer.Cube = domain.Cube{Owner: domain.None, Value: doubler.Cube.Value + 1}
	if !IsResponsePosition(&answer) {
		t.Fatal("a turned cube held by no one is not read as a reply")
	}
	if IsResponsePosition(&doubler) {
		t.Fatal("the doubler's own decision is read as a reply")
	}
	if got := DoublerPosition(answer); got.PlayerOnRoll != doubler.PlayerOnRoll || got.Cube != doubler.Cube {
		t.Fatalf("DoublerPosition = on roll %d, cube %+v; want %d, %+v", got.PlayerOnRoll, got.Cube, doubler.PlayerOnRoll, doubler.Cube)
	}

	direct, err := EvaluatePositionWithMET(nil, doubler, nil, 0, 0, 0)
	if err != nil || direct.Cube == nil {
		t.Fatalf("direct: %v", err)
	}
	reply, err := EvaluateResponseWithMET(nil, answer, nil, 0, 0)
	if err != nil || reply.Cube == nil {
		t.Fatalf("reply: %v", err)
	}
	d, r := direct.Cube, reply.Cube
	if r.CubefulNoDoubleEquity != d.CubefulNoDoubleEquity || r.CubefulDoubleTakeEquity != d.CubefulDoubleTakeEquity ||
		r.CubefulDoublePassEquity != d.CubefulDoublePassEquity || r.BestCubeAction != d.BestCubeAction {
		t.Errorf("reply equities %+v, want the doubler's %+v", r, d)
	}
	if r.PlayerWinChances != d.OpponentWinChances || r.OpponentGammonChances != d.PlayerGammonChances ||
		r.PlayerBackgammonChances != d.OpponentBackgammonChances {
		t.Errorf("reply chances not turned to the answerer: %+v vs %+v", r, d)
	}
}
