package sqlite_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// openMem returns a fresh in-memory SQLite Storage with the v2.7.0 schema.
func openMem(t *testing.T) *sqlite.Storage {
	t.Helper()
	s, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// TestMatchUpdate covers Update (with date parsing) and UpdateComment — paths
// the shared contract suite does not exercise.
func TestMatchUpdate(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob"}
	id, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := s.Matches().Update(ctx, "", id, "Anna", "Ben", "2024-12-25"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if err := s.Matches().UpdateComment(ctx, "", id, "edited"); err != nil {
		t.Fatalf("UpdateComment: %v", err)
	}

	got, err := s.Matches().Get(ctx, "", id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Player1Name != "Anna" || got.Player2Name != "Ben" {
		t.Errorf("Update players: got %q/%q", got.Player1Name, got.Player2Name)
	}
	if got.Comment != "edited" {
		t.Errorf("UpdateComment: got %q", got.Comment)
	}
	if got.MatchDate.Year() != 2024 || got.MatchDate.Month() != 12 || got.MatchDate.Day() != 25 {
		t.Errorf("Update date: got %v", got.MatchDate)
	}

	if err := s.Matches().Update(ctx, "", id, "x", "y", "not-a-date"); err == nil {
		t.Error("Update with bad date: expected error")
	}
}

// TestMatchSwapPlayers checks the header / game / move swap.
func TestMatchSwapPlayers(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob"}
	matchID, _ := s.Matches().Save(ctx, "", &m)
	g := domain.Game{MatchID: matchID, GameNumber: 1, InitialScore: [2]int32{2, 5}, Winner: 1}
	gameID, _ := s.Matches().CreateGame(ctx, "", &g)
	mv := domain.Move{GameID: gameID, MoveNumber: 1, MoveType: "checker", Player: 1}
	if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
		t.Fatalf("CreateMove: %v", err)
	}

	if err := s.Matches().SwapPlayers(ctx, "", matchID); err != nil {
		t.Fatalf("SwapPlayers: %v", err)
	}

	got, _ := s.Matches().Get(ctx, "", matchID)
	if got.Player1Name != "Bob" || got.Player2Name != "Alice" {
		t.Errorf("swap names: got %q/%q", got.Player1Name, got.Player2Name)
	}
	for g, err := range s.Matches().Games(ctx, "", matchID) {
		if err != nil {
			t.Fatalf("Games: %v", err)
		}
		if g.InitialScore != [2]int32{5, 2} || g.Winner != -1 {
			t.Errorf("swap game: score=%v winner=%d", g.InitialScore, g.Winner)
		}
	}
	for mv, err := range s.Matches().Moves(ctx, "", gameID) {
		if err != nil {
			t.Fatalf("Moves: %v", err)
		}
		if mv.Player != -1 {
			t.Errorf("swap move player: got %d, want -1", mv.Player)
		}
	}
}

// TestMatchMergePlayers checks the canonical-name rewrite.
func TestMatchMergePlayers(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	for _, p := range []string{"Bob", "Bobby", "Robert"} {
		mm := domain.Match{Player1Name: p, Player2Name: "Opponent"}
		if _, err := s.Matches().Save(ctx, "", &mm); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}
	mm := domain.Match{Player1Name: "Opponent", Player2Name: "Bobby"}
	if _, err := s.Matches().Save(ctx, "", &mm); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := s.Matches().MergePlayers(ctx, "", []string{"Bob", "Bobby", "Robert"}, "Bob R."); err != nil {
		t.Fatalf("MergePlayers: %v", err)
	}

	for got, err := range s.Matches().List(ctx, "") {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		switch got.Player1Name {
		case "Bob", "Bobby", "Robert":
			t.Errorf("player1 not merged: %q", got.Player1Name)
		}
		if got.Player2Name == "Bobby" {
			t.Errorf("player2 not merged: %q", got.Player2Name)
		}
	}

	if err := s.Matches().MergePlayers(ctx, "", nil, "X"); err == nil {
		t.Error("MergePlayers with no names: expected error")
	}
	if err := s.Matches().MergePlayers(ctx, "", []string{"a"}, ""); err == nil {
		t.Error("MergePlayers with empty canonical: expected error")
	}
}

// TestMatchLastVisited covers SetLastVisitedPosition and the LastVisited
// visited-first / recent-fallback / ErrNotFound paths.
func TestMatchLastVisited(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	if _, err := s.Matches().LastVisited(ctx, ""); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("LastVisited on empty db: got %v, want ErrNotFound", err)
	}

	m1 := domain.Match{Player1Name: "A1", Player2Name: "B1"}
	id1, _ := s.Matches().Save(ctx, "", &m1)
	m2 := domain.Match{Player1Name: "A2", Player2Name: "B2"}
	if _, err := s.Matches().Save(ctx, "", &m2); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := s.Matches().LastVisited(ctx, ""); err != nil {
		t.Fatalf("LastVisited fallback: %v", err)
	}

	if err := s.Matches().SetLastVisitedPosition(ctx, "", id1, 4); err != nil {
		t.Fatalf("SetLastVisitedPosition: %v", err)
	}
	lv, err := s.Matches().LastVisited(ctx, "")
	if err != nil {
		t.Fatalf("LastVisited: %v", err)
	}
	if lv.ID != id1 || lv.LastVisitedPosition != 4 {
		t.Errorf("LastVisited: got match %d pos %d, want %d pos 4", lv.ID, lv.LastVisitedPosition, id1)
	}
}

// TestMatchMovesByMatch checks the one-pass, game-then-move ordered stream of a
// multi-game match (single-tenant SQLite backend, Desktop path).
func TestMatchMovesByMatch(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob"}
	matchID, _ := s.Matches().Save(ctx, "", &m)
	g1 := domain.Game{MatchID: matchID, GameNumber: 1}
	g1ID, _ := s.Matches().CreateGame(ctx, "", &g1)
	g2 := domain.Game{MatchID: matchID, GameNumber: 2}
	g2ID, _ := s.Matches().CreateGame(ctx, "", &g2)

	// Insert across games and out of move order to prove the ORDER BY.
	for _, mv := range []domain.Move{
		{GameID: g2ID, MoveNumber: 1, MoveType: "checker", CheckerMove: "g2m1"},
		{GameID: g1ID, MoveNumber: 2, MoveType: "checker", CheckerMove: "g1m2"},
		{GameID: g1ID, MoveNumber: 1, MoveType: "checker", CheckerMove: "g1m1"},
	} {
		mv := mv
		if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove: %v", err)
		}
	}

	var moves []domain.Move
	for mv, err := range s.Matches().MovesByMatch(ctx, "", matchID) {
		if err != nil {
			t.Fatalf("MovesByMatch: %v", err)
		}
		moves = append(moves, *mv)
	}
	if len(moves) != 3 {
		t.Fatalf("MovesByMatch count: got %d, want 3", len(moves))
	}
	want := []struct {
		game int64
		cm   string
	}{{g1ID, "g1m1"}, {g1ID, "g1m2"}, {g2ID, "g2m1"}}
	for i, w := range want {
		if moves[i].GameID != w.game || moves[i].CheckerMove != w.cm {
			t.Errorf("move %d: got game=%d cm=%q, want game=%d cm=%q",
				i, moves[i].GameID, moves[i].CheckerMove, w.game, w.cm)
		}
	}

	// Unknown match → empty stream, no error.
	n := 0
	for _, err := range s.Matches().MovesByMatch(ctx, "", 9999) {
		if err != nil {
			t.Fatalf("MovesByMatch missing: %v", err)
		}
		n++
	}
	if n != 0 {
		t.Errorf("MovesByMatch on missing match: got %d moves, want 0", n)
	}
}
