package gammonnet

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync"
	"testing"
)

// scoreMapChecker: the opening 3-1 at 3-away/5-away in a 9-point match.
const scoreMapCheckerXGID = "XGID=-b----E-C---eE---c-e----B-:0:0:1:31:6:4:0:9:10"

func TestScoreMap_CheckerCells(t *testing.T) {
	pos := decodeXGIDForTest(t, scoreMapCheckerXGID)
	m, err := ComputeScoreMap(context.Background(), pos, nil, 5, 0, 0, LocalExec(4, 0, 0), nil)
	if err != nil {
		t.Fatal(err)
	}
	if m.Decision != "checker" || len(m.Cells) != 25 {
		t.Fatalf("decision %q, %d cells", m.Decision, len(m.Cells))
	}
	for idx, c := range m.Cells {
		if c.AwayOnRoll != idx/5+1 || c.AwayOpponent != idx%5+1 {
			t.Fatalf("cell %d holds %d/%d", idx, c.AwayOnRoll, c.AwayOpponent)
		}
		if c.Refused {
			t.Fatalf("cell %d/%d refused: %s", c.AwayOnRoll, c.AwayOpponent, c.Reason)
		}
		if c.Move == "" || c.SecondMove == "" || c.Gap == nil || *c.Gap < 0 {
			t.Fatalf("cell %d/%d: %+v", c.AwayOnRoll, c.AwayOpponent, c)
		}
		if math.Abs(*c.Gap-(c.Equity-*c.SecondEquity)) > 1e-12 {
			t.Fatalf("cell %d/%d: gap is not best minus second", c.AwayOnRoll, c.AwayOpponent)
		}
	}
}

// A cell is the Eval panel's answer at that score: same best play, same
// equity.
func TestScoreMap_AgreesWithASingleEvaluation(t *testing.T) {
	pos := decodeXGIDForTest(t, scoreMapCheckerXGID)
	m, err := ComputeScoreMap(context.Background(), pos, nil, 5, 0, 0, LocalExec(2, 0, 0), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct{ onRoll, opponent int }{{2, 3}, {4, 2}, {5, 5}} {
		single := pos
		single.Score = scoreForCell(pos.PlayerOnRoll, want.onRoll, want.opponent)
		res, err := EvaluatePosition(single, 0, 0, 2)
		if err != nil {
			t.Fatal(err)
		}
		c := m.Cells[(want.onRoll-1)*5+want.opponent-1]
		if c.Move != res.Moves[0].Move || math.Abs(c.Equity-res.Moves[0].Equity) > 1e-12 {
			t.Errorf("%d/%d: map %s %.6f, panel %s %.6f", want.onRoll, want.opponent, c.Move, c.Equity, res.Moves[0].Move, res.Moves[0].Equity)
		}
	}
}

// Without dice, the map is the cube matrix.
func TestScoreMap_CubeIsTheCubeMatrix(t *testing.T) {
	pos := cubeMatrixPosition(t)
	m, err := ComputeScoreMap(context.Background(), pos, nil, 5, 0, 0, LocalExec(2, 0, 0), nil)
	if err != nil {
		t.Fatal(err)
	}
	cm, err := ComputeCubeMatrix(context.Background(), pos, 5, 0, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if m.Decision != "cube" {
		t.Fatalf("decision %q", m.Decision)
	}
	for i := range cm.Cells {
		if m.Cells[i].CubeMatrixCell != cm.Cells[i] {
			t.Fatalf("cell %d: map %+v, matrix %+v", i, m.Cells[i].CubeMatrixCell, cm.Cells[i])
		}
	}
}

// The position's cube is kept; where it cannot stand, the cell says so.
func TestScoreMap_UnreachableCubeIsRefused(t *testing.T) {
	pos := decodeXGIDForTest(t, scoreMapCheckerXGID)
	pos.Cube.Value = 2 // a 4-cube
	pos.Cube.Owner = pos.PlayerOnRoll
	m, err := ComputeScoreMap(context.Background(), pos, nil, 5, 0, 0, LocalExec(2, 0, 0), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range m.Cells {
		// The opponent doubled from 2: pointless once 2 points win them the match.
		wantRefused := c.AwayOpponent <= 2
		if c.Refused != wantRefused {
			t.Errorf("%d/%d: refused %v (%s), want %v", c.AwayOnRoll, c.AwayOpponent, c.Refused, c.Reason, wantRefused)
		}
		if c.Refused && !strings.Contains(c.Reason, ErrNotEvaluable.Error()) {
			t.Errorf("%d/%d: reason %q does not name a refusal", c.AwayOnRoll, c.AwayOpponent, c.Reason)
		}
	}
}

// The refinement starts at the position's own score and moves outwards.
func TestScoreMap_RefinesFromThePositionsScore(t *testing.T) {
	pos := decodeXGIDForTest(t, scoreMapCheckerXGID)
	state, ok := MatchStateFromPosition(&pos)
	if !ok {
		t.Fatal("no match state")
	}
	var mu sync.Mutex
	var deep []ScoreMapCell
	_, err := ComputeScoreMap(context.Background(), pos, nil, 7, 1, 0, LocalExec(1, 1, 0), func(_ int, c ScoreMapCell) {
		mu.Lock()
		defer mu.Unlock()
		if c.Ply == 1 {
			deep = append(deep, c)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(deep) != 49 {
		t.Fatalf("%d cells refined, want 49", len(deep))
	}
	if deep[0].AwayOnRoll != state.AwayOnRoll || deep[0].AwayOpponent != state.AwayOpponent {
		t.Fatalf("first refined %d/%d, want the position's %d/%d", deep[0].AwayOnRoll, deep[0].AwayOpponent, state.AwayOnRoll, state.AwayOpponent)
	}
	dist := func(c ScoreMapCell) int {
		return abs(c.AwayOnRoll-state.AwayOnRoll) + abs(c.AwayOpponent-state.AwayOpponent)
	}
	for i := 1; i < len(deep); i++ {
		if dist(deep[i]) < dist(deep[i-1]) {
			t.Fatalf("cell %d (%d/%d) refined after a farther one", i, deep[i].AwayOnRoll, deep[i].AwayOpponent)
		}
	}
}

func TestScoreMap_Refusals(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	pos := decodeXGIDForTest(t, scoreMapCheckerXGID)
	if _, err := ComputeScoreMap(ctx, pos, nil, 9, 2, 0, LocalExec(2, 2, 0), nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled: err = %v", err)
	}
	if _, err := ComputeScoreMap(context.Background(), pos, nil, 0, 0, 0, LocalExec(2, 0, 0), nil); err == nil {
		t.Fatal("match length 0 accepted")
	}
	dance := decodeXGIDForTest(t, allDanceXGID)
	dance.Dice = [2]int{6, 6}
	if _, err := ComputeScoreMap(context.Background(), dance, nil, 5, 0, 0, LocalExec(2, 0, 0), nil); !errors.Is(err, ErrNotEvaluable) {
		t.Fatalf("a dance: err = %v", err)
	}
}
