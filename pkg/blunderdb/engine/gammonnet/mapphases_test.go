package gammonnet

import (
	"context"
	"errors"
	"math"
	"slices"
	"sync"
	"testing"
)

// busyMatchXGID: a middle game at 9 points, the cube centred, no dice.
const busyMatchXGID = "XGID=---BBaB-BbA-bC-b--BdAca---:0:0:1:00:0:5:0:9:10"

func TestMapDepths(t *testing.T) {
	for ply, want := range map[int][]int{0: {0}, 1: {0, 1}, 2: {0, 1, 2}, 3: {0, 1, 3}} {
		if got := mapDepths(ply); !slices.Equal(got, want) {
			t.Errorf("mapDepths(%d) = %v, want %v", ply, got, want)
		}
	}
}

// Every row arrives at 0, then 1, then 2 ply, never going back.
func TestRollMap_ThreeTiers(t *testing.T) {
	pos := decodeXGIDForTest(t, openingXGID)
	var mu sync.Mutex
	plies := map[int][]int{}
	_, err := ComputeRollMap(context.Background(), pos, nil, 2, 0, LocalExec(0, 2, 0), func(i int, r RollMapRow) {
		mu.Lock()
		defer mu.Unlock()
		plies[i] = append(plies[i], r.Ply)
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := range NumRolls {
		if !slices.Equal(plies[i], []int{0, 1, 2}) {
			t.Fatalf("row %d arrived at plies %v, want [0 1 2]", i, plies[i])
		}
	}
}

// oracleCases cover a centred and an owned cube, the filters off and on.
var oracleCases = []struct {
	xgid   string
	pruneK int
}{
	{openingXGID, 0},
	{busyMatchXGID, 0},
	{ownedCubeMatchXGID, DefaultPruneK},
	{ownedCubeMoneyXGID, DefaultPruneK},
}

// ownedCubeMatchXGID: busyMatchXGID with a 2-cube the player on roll owns.
const ownedCubeMatchXGID = "XGID=---BBaB-BbA-bC-b--BdAca---:1:1:1:00:0:5:0:9:10"

// ownedCubeMoneyXGID: the opening at money, a 2-cube the opponent owns.
const ownedCubeMoneyXGID = "XGID=-b----E-C---eE---c-e----B-:1:-1:1:00:0:0:0:0:10"

// The provisional phases and the shared searchers leave no trace: every final
// row is, bit for bit, the row searched alone on a fresh serial searcher.
func TestRollMap_FinalRowsAreTheRowsSearchedAlone(t *testing.T) {
	for _, c := range oracleCases {
		pos := decodeXGIDForTest(t, c.xgid)
		m, err := ComputeRollMap(context.Background(), pos, nil, 2, c.pruneK, LocalExec(0, 2, c.pruneK), nil)
		if err != nil {
			t.Fatal(err)
		}
		job, err := newRollMapJob(pos, nil, 2, c.pruneK)
		if err != nil {
			t.Fatal(err)
		}
		var avg float64
		for i := range NumRolls {
			s, err := NewSearcher(job.cfg)
			if err != nil {
				t.Fatal(err)
			}
			want, err := job.row(context.Background(), i, s, c.pruneK)
			if err != nil {
				t.Fatal(err)
			}
			got := m.Rows[i]
			if math.Float64bits(got.Equity) != math.Float64bits(want.Equity) || got.Move != want.Move ||
				got.Dance != want.Dance || got.Ply != want.Ply || !equalBoolPtr(got.MarketLost, want.MarketLost) {
				t.Fatalf("%s pruneK %d row %d = %+v, alone %+v", c.xgid, c.pruneK, i, got, want)
			}
			avg += float64(job.rolls[i].weight * want.Equity)
		}
		if math.Float64bits(m.Average) != math.Float64bits(avg) {
			t.Fatalf("%s pruneK %d average %v, alone %v", c.xgid, c.pruneK, m.Average, avg)
		}
	}
}

func equalBoolPtr(a, b *bool) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// The score map's final cells, checker and cube, centred and owned, filters
// off and on, are the cells searched alone.
func TestScoreMap_FinalCellsAreTheCellsSearchedAlone(t *testing.T) {
	cases := []struct {
		xgid   string
		pruneK int
	}{
		{scoreMapCheckerXGID, 0},
		{busyMatchXGID, 0},
		{ownedCubeMatchXGID, DefaultPruneK},
		{"XGID=-b----E-C---eE---c-e----B-:1:1:1:31:6:4:0:9:10", DefaultPruneK},
	}
	for _, c := range cases {
		pos := decodeXGIDForTest(t, c.xgid)
		const length = 5
		m, err := ComputeScoreMap(context.Background(), pos, nil, length, 2, c.pruneK, LocalExec(0, 2, c.pruneK), nil)
		if err != nil {
			t.Fatal(err)
		}
		alone, err := ComputeScoreMap(context.Background(), pos, nil, length, 2, c.pruneK, aloneExec, nil)
		if err != nil {
			t.Fatal(err)
		}
		for i := range m.Cells {
			got, want := m.Cells[i], alone.Cells[i]
			if got.Ply != 2 || !sameScoreMapCell(got, want) {
				t.Fatalf("%s pruneK %d cell %d = %+v, alone %+v", c.xgid, c.pruneK, i, got, want)
			}
		}
	}
}

// aloneExec runs the cells one after another, each on a searcher of its own
// (nil: the cell builds a NewBatchSearcher, which forks from 2 ply), so no
// cache and no earlier phase is shared between cells.
func aloneExec(n int, cell func(int, *Searcher)) error {
	for i := range n {
		cell(i, nil)
	}
	return nil
}

func sameScoreMapCell(a, b ScoreMapCell) bool {
	bits := math.Float64bits
	optBits := func(p, q *float64) bool {
		if p == nil || q == nil {
			return p == q
		}
		return bits(*p) == bits(*q)
	}
	return a.AwayOnRoll == b.AwayOnRoll && a.AwayOpponent == b.AwayOpponent && a.Verdict == b.Verdict &&
		bits(a.NoDouble) == bits(b.NoDouble) && bits(a.DoubleTake) == bits(b.DoubleTake) &&
		bits(a.DoublePass) == bits(b.DoublePass) && a.Refused == b.Refused && a.Reason == b.Reason &&
		a.Ply == b.Ply && a.Crawford == b.Crawford && a.Move == b.Move && bits(a.Equity) == bits(b.Equity) &&
		a.SecondMove == b.SecondMove && optBits(a.SecondEquity, b.SecondEquity) && optBits(a.Gap, b.Gap)
}

// A cancellation during the deep phase stops it between cells: no cell
// starts after it, so at most the cells already searching can land.
func TestMaps_CancelStopsTheDeepPhaseBetweenCells(t *testing.T) {
	t.Run("rolls", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var mu sync.Mutex
		deep := 0
		_, err := ComputeRollMap(ctx, decodeXGIDForTest(t, openingXGID), nil, 2, 0, LocalExec(0, 2, 0), func(_ int, r RollMapRow) {
			mu.Lock()
			defer mu.Unlock()
			if r.Ply == 2 {
				deep++
				cancel()
			}
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
		if deep < 1 || deep > mapDeepWidth {
			t.Fatalf("%d rows landed at 2 ply after the cancellation, want 1 to %d", deep, mapDeepWidth)
		}
	})
	t.Run("scores", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var mu sync.Mutex
		deep := 0
		_, err := ComputeScoreMap(ctx, decodeXGIDForTest(t, scoreMapCheckerXGID), nil, 5, 2, 0, LocalExec(0, 2, 0), func(_ int, c ScoreMapCell) {
			mu.Lock()
			defer mu.Unlock()
			if c.Ply == 2 {
				deep++
				cancel()
			}
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
		if deep < 1 || deep > mapDeepWidth {
			t.Fatalf("%d cells landed at 2 ply after the cancellation, want 1 to %d", deep, mapDeepWidth)
		}
	})
}

// LocalExec runs a deep map's phase cell by cell, every cell once and never
// more than mapDeepWidth at once, so the cores left over go to the cells' own
// forks.
func TestLocalExec_DeepMapIsNarrow(t *testing.T) {
	var mu sync.Mutex
	calls := map[int]int{}
	running, peak := 0, 0
	err := runMapPhase(context.Background(), LocalExec(0, 2, 0), 30, func(i int, _ *Searcher) {
		mu.Lock()
		calls[i]++
		running++
		peak = max(peak, running)
		mu.Unlock()
		_, _ = NewBatchSearcher(0, 0) // a little work, for the units to overlap
		mu.Lock()
		running--
		mu.Unlock()
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 30 || peak > mapDeepWidth {
		t.Fatalf("%d cells called, %d at once (width %d)", len(calls), peak, mapDeepWidth)
	}
	for i, n := range calls {
		if n != 1 {
			t.Fatalf("cell %d called %d times", i, n)
		}
	}
}
