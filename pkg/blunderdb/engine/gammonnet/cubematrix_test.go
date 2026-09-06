package gammonnet

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// cubeMatrixPosition is a real cube decision: the doubling side is ahead in a
// straight race, so the verdict is expected to swing across the grid rather
// than repeat one answer.
func cubeMatrixPosition(t *testing.T) domain.Position {
	t.Helper()
	pos, err := domain.DecodeXGID("XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:7:10")
	if err != nil {
		t.Fatalf("DecodeXGID: %v", err)
	}
	return pos
}

func TestComputeCubeMatrix_ShapeAndScores(t *testing.T) {
	pos := cubeMatrixPosition(t)

	m, err := ComputeCubeMatrix(context.Background(), pos, 5, 0, 0, 4)
	if err != nil {
		t.Fatalf("ComputeCubeMatrix: %v", err)
	}
	if m.MatchLength != 5 {
		t.Fatalf("match length = %d, want 5", m.MatchLength)
	}
	if len(m.Cells) != 25 {
		t.Fatalf("cells = %d, want 25", len(m.Cells))
	}

	seen := map[[2]int]bool{}
	for _, c := range m.Cells {
		key := [2]int{c.AwayOnRoll, c.AwayOpponent}
		if seen[key] {
			t.Fatalf("duplicate cell %v", key)
		}
		seen[key] = true
		if c.AwayOnRoll < 1 || c.AwayOnRoll > 5 || c.AwayOpponent < 1 || c.AwayOpponent > 5 {
			t.Fatalf("cell out of range: %v", key)
		}
		if c.Refused {
			if c.Reason == "" {
				t.Fatalf("cell %v refused without a reason", key)
			}
			continue
		}
		switch c.Verdict {
		case "no_double", "double_take", "double_pass", "too_good":
		default:
			t.Fatalf("cell %v: unknown verdict %q", key, c.Verdict)
		}
		// Normalised equity: ±1 is winning or losing the current cube
		// (ADR-0019). A cell outside a small margin of that band is a scale
		// bug, the exact kind ADR-0019 exists to catch.
		for _, v := range []float64{c.NoDouble, c.DoubleTake, c.DoublePass} {
			if v < -3 || v > 3 {
				t.Fatalf("cell %v: equity %v is off the normalised scale", key, v)
			}
		}
	}
	if len(seen) != 25 {
		t.Fatalf("distinct cells = %d, want 25", len(seen))
	}
}

// The point of the whole feature: the verdict is a property of the SCORE, not
// only of the board. A grid that answered the same thing everywhere would
// have been cheaper to compute and worth nothing.
func TestComputeCubeMatrix_VerdictVariesWithScore(t *testing.T) {
	pos := cubeMatrixPosition(t)

	m, err := ComputeCubeMatrix(context.Background(), pos, 5, 0, 0, 4)
	if err != nil {
		t.Fatalf("ComputeCubeMatrix: %v", err)
	}
	verdicts := map[string]int{}
	for _, c := range m.Cells {
		if !c.Refused {
			verdicts[c.Verdict]++
		}
	}
	if len(verdicts) < 2 {
		t.Fatalf("the grid says %v at every score; expected the score to matter", verdicts)
	}
	t.Logf("verdicts across the 5-point grid: %v", verdicts)
}

// The grid is independent of how many goroutines computed it: cells share
// nothing, and each one runs its own search. This is the same guarantee the
// analysis batch makes (CLAUDE.md, "Parallelism is production behaviour").
func TestComputeCubeMatrix_WorkerCountDoesNotChangeIt(t *testing.T) {
	pos := cubeMatrixPosition(t)

	one, err := ComputeCubeMatrix(context.Background(), pos, 5, 0, 0, 1)
	if err != nil {
		t.Fatalf("serial: %v", err)
	}
	many, err := ComputeCubeMatrix(context.Background(), pos, 5, 0, 0, 8)
	if err != nil {
		t.Fatalf("parallel: %v", err)
	}
	for i := range one.Cells {
		if one.Cells[i] != many.Cells[i] {
			t.Fatalf("cell %d differs:\n serial   %+v\n parallel %+v", i, one.Cells[i], many.Cells[i])
		}
	}
}

func TestComputeCubeMatrix_RefusesAnEmptyMatchLength(t *testing.T) {
	pos := cubeMatrixPosition(t)
	if _, err := ComputeCubeMatrix(context.Background(), pos, 0, 0, 0, 1); err == nil {
		t.Fatal("a 0-point match should be refused")
	}
}

// A cancelled sweep returns what it had rather than a torn grid: the caller
// gets ctx.Err() and can tell the difference.
func TestComputeCubeMatrix_HonoursCancellation(t *testing.T) {
	pos := cubeMatrixPosition(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ComputeCubeMatrix(ctx, pos, 9, 0, 0, 4)
	if err == nil {
		t.Fatal("a cancelled sweep should report its cancellation")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("err = %v, want a cancellation", err)
	}
}

// Not an assertion, a measurement: the fiche budgets the 2-ply sweep and the
// interface has to choose between running it on the gesture and running it at
// rest. -v prints the number that decides.
func TestComputeCubeMatrix_Cost(t *testing.T) {
	if testing.Short() {
		t.Skip("measurement, not a check")
	}
	pos := cubeMatrixPosition(t)

	for _, ply := range []int{0, 2} {
		for _, length := range CubeMatrixLengths {
			start := time.Now()
			m, err := ComputeCubeMatrix(context.Background(), pos, length, ply, 0, 0)
			if err != nil {
				t.Fatalf("%d-ply, %d points: %v", ply, length, err)
			}
			t.Logf("%d-ply, %d-point grid (%d cells): %v", ply, length, len(m.Cells), time.Since(start).Round(time.Millisecond))
		}
	}
}

// The grid and the Eval panel must not answer differently for the same score:
// one cell, taken on its own through the ordinary EvaluatePosition path, is
// the cell the sweep produced. This is the check that the sweep's score
// plumbing — away scores are stored BY PLAYER, and a raw 1 is the Crawford
// sentinel — actually lands where it means to.
func TestComputeCubeMatrix_AgreesWithASingleEvaluation(t *testing.T) {
	pos := cubeMatrixPosition(t)

	m, err := ComputeCubeMatrix(context.Background(), pos, 5, 0, 0, 4)
	if err != nil {
		t.Fatalf("ComputeCubeMatrix: %v", err)
	}

	for _, want := range []struct{ onRoll, opponent int }{{2, 3}, {4, 2}, {5, 5}} {
		single := pos
		single.Dice = [2]int{0, 0}
		single.Score = scoreForCell(pos.PlayerOnRoll, want.onRoll, want.opponent)
		single.DecisionType = domain.CubeAction

		res, err := EvaluatePosition(single, 0, 0, 0)
		if err != nil {
			t.Fatalf("%d-away/%d-away: %v", want.onRoll, want.opponent, err)
		}
		if res.Cube == nil {
			t.Fatalf("%d-away/%d-away: no cube analysis", want.onRoll, want.opponent)
		}

		var cell CubeMatrixCell
		for _, c := range m.Cells {
			if c.AwayOnRoll == want.onRoll && c.AwayOpponent == want.opponent {
				cell = c
			}
		}
		if cell.Refused {
			t.Fatalf("%d-away/%d-away: the grid refused a cell a single evaluation accepts", want.onRoll, want.opponent)
		}
		if got, exp := cell.NoDouble, res.Cube.CubefulNoDoubleEquity; got != exp {
			t.Errorf("%d-away/%d-away no-double: grid %v, single evaluation %v", want.onRoll, want.opponent, got, exp)
		}
		if got, exp := cell.DoubleTake, res.Cube.CubefulDoubleTakeEquity; got != exp {
			t.Errorf("%d-away/%d-away double/take: grid %v, single evaluation %v", want.onRoll, want.opponent, got, exp)
		}
	}
}
