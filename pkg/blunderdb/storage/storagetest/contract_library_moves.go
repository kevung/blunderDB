// Contract case for LibraryMoves, the one-pass read of every recorded move
// of a library. The table that runs it lives in contract.go.
package storagetest

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testMatchLibraryMoves pins LibraryMoves to the walk it replaces — List, then
// each match's Games, then each game's RecordedMoves — on a library whose
// insertion order is none of the orders read: matches saved oldest first and
// one undated, games and moves created out of number order, two games of a
// match numbered alike.
func testMatchLibraryMoves(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := s.Matches()

	pos := statsDecisionPos(t, 2)
	posID, err := s.Positions().Save(ctx, "", &pos)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	luck := int32(-42)
	ms5 := int64(1500)
	seed := func(m domain.Match) {
		t.Helper()
		matchID, err := ms.Save(ctx, "", &m)
		if err != nil {
			t.Fatalf("Save match: %v", err)
		}
		for _, gn := range []int32{2, 1, 2} {
			gameID, err := ms.CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: gn, Winner: 1, PointsWon: 1})
			if err != nil {
				t.Fatalf("CreateGame: %v", err)
			}
			for _, n := range []int32{3, 1, 2} {
				mv := domain.Move{GameID: gameID, MoveNumber: n, MoveType: "checker", PositionID: posID,
					Player: n % 2, Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5"}
				if n == 2 {
					mv.MoveType, mv.CheckerMove, mv.CubeAction = "cube", "", "Double"
					mv.Dice = [2]int32{}
				}
				if n == 1 {
					mv.LuckMP, mv.DecisionMS = &luck, &ms5
				}
				if _, err := ms.CreateMove(ctx, "", &mv); err != nil {
					t.Fatalf("CreateMove: %v", err)
				}
			}
		}
	}
	seed(domain.Match{Player1Name: "Old", Player2Name: "O", MatchLength: 5, MatchDate: time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)})
	seed(domain.Match{Player1Name: "New", Player2Name: "N", MatchLength: 7, MatchDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)})
	seed(domain.Match{Player1Name: "Undated", Player2Name: "U", MatchLength: 3})

	var want []domain.LibraryMove
	var matches []*domain.Match
	for m, err := range ms.List(ctx, "", storage.MatchListOpts{}) {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		matches = append(matches, m)
	}
	for _, m := range matches {
		var games []*domain.Game
		for g, err := range ms.Games(ctx, "", m.ID) {
			if err != nil {
				t.Fatalf("Games: %v", err)
			}
			games = append(games, g)
		}
		for _, g := range games {
			for mv, err := range ms.RecordedMoves(ctx, "", g.ID) {
				if err != nil {
					t.Fatalf("RecordedMoves: %v", err)
				}
				want = append(want, domain.LibraryMove{Move: *mv, MatchID: m.ID, GameNumber: g.GameNumber})
			}
		}
	}
	if len(want) != 27 {
		t.Fatalf("walk read %d moves, want 27", len(want))
	}

	var got []domain.LibraryMove
	for lm, err := range ms.LibraryMoves(ctx, "") {
		if err != nil {
			t.Fatalf("LibraryMoves: %v", err)
		}
		got = append(got, *lm)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("LibraryMoves differs from List → Games → RecordedMoves:\n got %+v\nwant %+v", got, want)
	}

	// Stopping early stops the stream without error.
	n := 0
	for _, err := range ms.LibraryMoves(ctx, "") {
		if err != nil {
			t.Fatalf("LibraryMoves: %v", err)
		}
		n++
		if n == 4 {
			break
		}
	}
}
