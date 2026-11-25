package gammonnet

import (
	"context"
	"errors"
	"math"
	"sync"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// allDanceXGID: the player on roll is on the bar against a closed board.
const allDanceXGID = "XGID=bBBBBBB-C----e-----e--c---:0:0:-1:00:0:0:0:0:10"

func decodeXGIDForTest(t *testing.T, xgid string) domain.Position {
	t.Helper()
	pos, err := domain.DecodeXGID(xgid)
	if err != nil {
		t.Fatalf("DecodeXGID(%s): %v", xgid, err)
	}
	return pos
}

// parentEquity is the position's own search one ply deeper than the map, in
// the same referential: the number the map must decompose.
func parentEquity(t *testing.T, pos domain.Position, depth int) float64 {
	t.Helper()
	pos.Dice = [2]int{0, 0}
	cfg, state, err := ConfigForPositionMET(&pos, nil, depth, 0)
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSearcher(cfg)
	if err != nil {
		t.Fatal(err)
	}
	gn, err := FromDomain(&pos)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := s.positionEquity(&gn, depth, 0, state, cfg.CubeOwner)
	if !ok {
		t.Fatal("positionEquity failed")
	}
	scale, _ := NewEquityScale(state)
	return scale.FromSearch(v)
}

func TestRollMap_AverageIsTheParentOnePlyDeeper(t *testing.T) {
	cases := map[string]string{
		"money opening": openingXGID,
		"match, busy":   busyXGID,
		"all dances":    allDanceXGID,
	}
	for name, xgid := range cases {
		for _, ply := range []int{0, 1} {
			pos := decodeXGIDForTest(t, xgid)
			m, err := ComputeRollMap(context.Background(), pos, nil, ply, 0, LocalExec(4, ply, 0), nil)
			if err != nil {
				t.Fatalf("%s %d-ply: %v", name, ply, err)
			}
			want := parentEquity(t, pos, ply+1)
			if math.Abs(m.Average-want) > 1e-9 {
				t.Errorf("%s %d-ply: average %.12f, parent at %d-ply %.12f", name, ply, m.Average, ply+1, want)
			}
		}
	}
}

func TestRollMap_RowsAndWeights(t *testing.T) {
	pos := decodeXGIDForTest(t, openingXGID)
	m, err := ComputeRollMap(context.Background(), pos, nil, 0, 0, LocalExec(2, 0, 0), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Rows) != NumRolls {
		t.Fatalf("rows = %d, want %d", len(m.Rows), NumRolls)
	}
	total := 0
	for _, r := range m.Rows {
		total += r.Weight
		if r.Die1 > r.Die2 || (r.Die1 == r.Die2) != (r.Weight == 1) {
			t.Errorf("row %d-%d has weight %d", r.Die1, r.Die2, r.Weight)
		}
		if r.Move == "" || r.Dance {
			t.Errorf("row %d-%d: no play in the opening", r.Die1, r.Die2)
		}
		if !m.CubeAccess || r.MarketLost == nil {
			t.Errorf("row %d-%d: centred cube, market missing", r.Die1, r.Die2)
		} else if *r.MarketLost {
			t.Errorf("row %d-%d: no opening roll loses the market", r.Die1, r.Die2)
		}
	}
	if total != 36 {
		t.Fatalf("weights sum to %d, want 36", total)
	}
}

func TestRollMap_DancesAndCubeAccess(t *testing.T) {
	pos := decodeXGIDForTest(t, allDanceXGID)
	m, err := ComputeRollMap(context.Background(), pos, nil, 0, 0, LocalExec(2, 0, 0), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range m.Rows {
		if !r.Dance || r.Move != "" {
			t.Errorf("row %d-%d: want a dance, got %+v", r.Die1, r.Die2, r)
		}
	}

	// The opponent holds the cube: nobody asks the roller's market.
	pos.Cube.Value = 1
	pos.Cube.Owner = opponentOf(pos.PlayerOnRoll)
	m, err = ComputeRollMap(context.Background(), pos, nil, 0, 0, LocalExec(2, 0, 0), nil)
	if err != nil {
		t.Fatal(err)
	}
	if m.CubeAccess {
		t.Fatal("cube access with the opponent's cube")
	}
	for _, r := range m.Rows {
		if r.MarketLost != nil {
			t.Fatalf("row %d-%d: market reported without cube access", r.Die1, r.Die2)
		}
	}
}

func TestRollMap_ProgressiveRows(t *testing.T) {
	pos := decodeXGIDForTest(t, openingXGID)
	var mu sync.Mutex
	seen := map[int]int{}
	_, err := ComputeRollMap(context.Background(), pos, nil, 1, 0, LocalExec(4, 1, 0), func(i int, r RollMapRow) {
		mu.Lock()
		defer mu.Unlock()
		seen[r.Ply]++
	})
	if err != nil {
		t.Fatal(err)
	}
	if seen[0] != NumRolls || seen[1] != NumRolls {
		t.Fatalf("rows per depth = %v, want %d at 0 and 1", seen, NumRolls)
	}
}

func TestRollMap_Cancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ComputeRollMap(ctx, decodeXGIDForTest(t, openingXGID), nil, 2, 0, LocalExec(2, 2, 0), nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestPositionAfterPlay_HandsTheTurnOver(t *testing.T) {
	pos := decodeXGIDForTest(t, openingXGID)
	after, err := PositionAfterPlay(pos, 3, 1, "8/5 6/5")
	if err != nil {
		t.Fatal(err)
	}
	if after.PlayerOnRoll == pos.PlayerOnRoll || after.Dice != [2]int{0, 0} {
		t.Fatalf("turn or dice not handed over: %+v", after)
	}
	if _, err := PositionAfterPlay(pos, 3, 1, "24/18"); err == nil {
		t.Fatal("an illegal play was accepted")
	}
	if _, err := PositionAfterPlay(pos, 3, 1, ""); err == nil {
		t.Fatal("a missing play was accepted")
	}
	// The replies map runs on the result, the opponent on roll.
	if _, err := ComputeRollMap(context.Background(), after, nil, 0, 0, LocalExec(2, 0, 0), nil); err != nil {
		t.Fatal(err)
	}
}

func opponentOf(player int) int {
	if player == domain.White {
		return domain.Black
	}
	return domain.White
}

func TestRollMap_MarketLostWhenAlreadyTooGood(t *testing.T) {
	m, err := ComputeRollMap(context.Background(), decodeXGIDForTest(t, tooGoodXGID), nil, 0, 0, LocalExec(2, 0, 0), nil)
	if err != nil {
		t.Fatal(err)
	}
	lost := 0
	for _, r := range m.Rows {
		if r.MarketLost != nil && *r.MarketLost {
			lost += r.Weight
		}
	}
	if lost < 30 {
		t.Fatalf("market lost on %d/36, want nearly every roll", lost)
	}
}
