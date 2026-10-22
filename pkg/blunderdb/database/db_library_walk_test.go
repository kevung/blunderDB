package database

import (
	"database/sql"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
)

// EachLibraryMove reads what the per-match getters read, in their order.
func TestEachLibraryMove_IsThePerGameWalk(t *testing.T) {
	t.Parallel()
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

// An error from fn stops the walk on the page it came from, with the next
// page already being read: the walk returns that error, calls fn no more, and
// leaves no read running against the database it hands back.
func TestEachPositionAnalysis_StopsOnFnError(t *testing.T) {
	db := newTestDBWithXG(t)
	saved := positionWalkPage
	positionWalkPage = 7
	t.Cleanup(func() { positionWalkPage = saved })

	stop := errors.New("stop")
	n := 0
	err := db.EachPositionAnalysis(func(*Position, *PositionAnalysis) error {
		n++
		if n == positionWalkPage+2 {
			return stop
		}
		return nil
	})
	if !errors.Is(err, stop) || n != positionWalkPage+2 {
		t.Errorf("err = %v after %d calls, want %v after %d", err, n, stop, positionWalkPage+2)
	}
}

// A panic in fn leaves no page read running once it has left the walk: the
// read of the next page, held until fn panics, has finished by the time the
// panic reaches the caller.
func TestEachPositionAnalysis_PanicLeavesNoReadInFlight(t *testing.T) {
	db := newTestDBWithXG(t)
	savedPage, savedRead := positionWalkPage, readWalkPage
	positionWalkPage = 7
	panicking := make(chan struct{})
	var started, finished atomic.Int32
	readWalkPage = func(d *Database, offset int) ([]Position, map[int64]*PositionAnalysis, error) {
		if offset > 0 {
			started.Add(1)
			<-panicking
			defer finished.Add(1)
		}
		return savedRead(d, offset)
	}
	t.Cleanup(func() { positionWalkPage, readWalkPage = savedPage, savedRead })

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("the panic of fn did not leave the walk")
			}
		}()
		_ = db.EachPositionAnalysis(func(*Position, *PositionAnalysis) error {
			close(panicking)
			panic("fn")
		})
	}()
	if s, f := started.Load(), finished.Load(); s != 1 || f != 1 {
		t.Errorf("next-page reads: %d started, %d finished when the panic left the walk; want 1 and 1", s, f)
	}
}
