package database

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"
)

// EachLibraryMove reads what the per-match getters read, in their order.
func TestEachLibraryMove_IsThePerGameWalk(t *testing.T) {
	db := newTestDBWithXG(t)
	type row struct {
		matchID    int64
		gameNumber int32
		mv         Move
	}
	var want []row
	matches, err := db.GetAllMatches()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range matches {
		games, err := db.GetGamesByMatch(m.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, g := range games {
			moves, err := db.GetMovesByGame(g.ID)
			if err != nil {
				t.Fatal(err)
			}
			for _, mv := range moves {
				want = append(want, row{m.ID, g.GameNumber, mv})
			}
		}
	}
	if len(want) == 0 {
		t.Fatal("the fixture has no move")
	}
	var got []row
	if err := db.EachLibraryMove(func(m *Match, gameNumber int32, mv *Move) error {
		got = append(got, row{m.ID, gameNumber, *mv})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("EachLibraryMove read %d rows differing from the per-game walk's %d", len(got), len(want))
	}

	stop := errors.New("stop")
	n := 0
	if err := db.EachLibraryMove(func(*Match, int32, *Move) error { n++; return stop }); !errors.Is(err, stop) || n != 1 {
		t.Errorf("an error from fn: err = %v after %d calls, want stop after 1", err, n)
	}
}

// EachPositionAnalysis reads, page by page, what LoadAllPositions and a
// LoadAnalysis per position read.
func TestEachPositionAnalysis_IsLoadAnalysisPerPosition(t *testing.T) {
	db := newTestDBWithXG(t)
	saved := positionWalkPage
	positionWalkPage = 7
	t.Cleanup(func() { positionWalkPage = saved })

	positions, err := db.LoadAllPositions()
	if err != nil {
		t.Fatal(err)
	}
	if len(positions) <= positionWalkPage {
		t.Fatalf("the fixture has %d positions, too few for two pages", len(positions))
	}
	i, analysed := 0, 0
	if err := db.EachPositionAnalysis(func(pos *Position, a *PositionAnalysis) error {
		if i >= len(positions) || !reflect.DeepEqual(*pos, positions[i]) {
			t.Fatalf("position %d differs from LoadAllPositions's", i)
		}
		want, err := db.LoadAnalysis(pos.ID)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			if a != nil {
				t.Errorf("position %d: an analysis where LoadAnalysis has none", pos.ID)
			}
		case err != nil:
			t.Fatal(err)
		case !reflect.DeepEqual(a, want):
			t.Errorf("position %d: analysis differs from LoadAnalysis's\n got %+v\nwant %+v", pos.ID, a, want)
		default:
			analysed++
		}
		i++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if i != len(positions) || analysed == 0 {
		t.Errorf("walked %d positions (%d analysed), want %d with some analysed", i, analysed, len(positions))
	}
}
