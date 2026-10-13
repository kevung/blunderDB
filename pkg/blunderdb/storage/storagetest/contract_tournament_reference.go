// Contract case for the tournament's reference player: the Main player when
// one is designated and played in the tournament, recognised through their
// aliases; otherwise the player of the most matches. The badge of the
// tournament list and the review's default player follow the same rule.
package storagetest

import (
	"context"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// saveTournamentMatch saves a one-game match of the tournament where seat 1
// plays one decision per slot and seat 2 one decision after them.
func saveTournamentMatch(t *testing.T, s storage.Storage, tid int64, p1, p2 string, slots []int) {
	t.Helper()
	ctx := context.Background()
	m := domain.Match{Player1Name: p1, Player2Name: p2, MatchLength: 7,
		MatchDate: time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC)}
	id, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	g := domain.Game{MatchID: id, GameNumber: 1, Winner: 1, PointsWon: 1}
	gameID, err := s.Matches().CreateGame(ctx, "", &g)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	play := func(n, slot int, player int32) {
		pos := statsDecisionPos(t, slot)
		posID, err := s.Positions().Save(ctx, "", &pos)
		if err != nil {
			t.Fatalf("Save position (slot %d): %v", slot, err)
		}
		eqErr := 0.01 * float64(slot+1)
		a := domain.PositionAnalysis{PlayedMoves: []string{"13/11 24/23"},
			CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
				{Move: "8/6 6/4", Equity: 0.5}, {Move: "13/11 24/23", Equity: 0.5 - eqErr, EquityError: &eqErr}}}}
		if err := s.Analyses().Save(ctx, "", posID, &a); err != nil {
			t.Fatalf("Save analysis (slot %d): %v", slot, err)
		}
		mv := domain.Move{GameID: gameID, MoveNumber: int32(n + 1), MoveType: "checker",
			PositionID: posID, Player: player, CheckerMove: "13/11 24/23"}
		if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove: %v", err)
		}
	}
	for i, slot := range slots {
		play(i, slot, 1)
	}
	play(len(slots), slots[len(slots)-1]+1, -1)
	if err := s.Tournaments().AddMatch(ctx, "", tid, id); err != nil {
		t.Fatalf("AddMatch: %v", err)
	}
}

func testStatsTournamentReferenceIsMainPlayer(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	tid, err := s.Tournaments().Create(ctx, "", "Open", "2025-06-10", "")
	if err != nil {
		t.Fatalf("Create tournament: %v", err)
	}
	// Alice plays two matches; Bob one under his name and one under an
	// alias. The fallback counts raw names, so it picks Alice; as Main
	// player, Bob's two spellings are pooled.
	saveTournamentMatch(t, s, tid, "Alice", "Bob", []int{0, 1})
	saveTournamentMatch(t, s, tid, "Alice", "Carol", []int{3, 4})
	saveTournamentMatch(t, s, tid, "Bobby", "Dave", []int{6, 7})
	if err := s.Aliases().Set(ctx, "", storage.AliasPlayer, "Bobby", "Bob"); err != nil {
		t.Fatalf("Set alias: %v", err)
	}

	check := func(main, want string) {
		t.Helper()
		if err := s.LibrarySettings().SetMainPlayer(ctx, "", main); err != nil {
			t.Fatalf("SetMainPlayer(%q): %v", main, err)
		}
		badges, err := s.Stats().TournamentBadges(ctx, "")
		if err != nil {
			t.Fatalf("TournamentBadges: %v", err)
		}
		if got := badges[tid].RefPlayer; got != want {
			t.Errorf("Main player %q: badge RefPlayer = %q, want %q", main, got, want)
		}
		review, err := s.Stats().TournamentReview(ctx, "", tid, "")
		if err != nil {
			t.Fatalf("TournamentReview: %v", err)
		}
		if review.Player != want {
			t.Errorf("Main player %q: review Player = %q, want %q", main, review.Player, want)
		}
	}

	check("", "Alice")      // no Main player: the most matches
	check("Zed", "Alice")   // a Main player absent from the tournament: same
	check("Bob", "Bob")     // the Main player, whatever their match count
	check("Bobby", "Bob")   // designated by an alias, named canonically
	check("Carol", "Carol") // one match, fewer than Alice's two

	// The Main player's figures are theirs across spellings: Bob's single
	// decision as seat 2 plus Bobby's two as seat 1.
	if err := s.LibrarySettings().SetMainPlayer(ctx, "", "Bob"); err != nil {
		t.Fatalf("SetMainPlayer: %v", err)
	}
	badges, err := s.Stats().TournamentBadges(ctx, "")
	if err != nil {
		t.Fatalf("TournamentBadges: %v", err)
	}
	// Errors are 0.01·(slot+1): slot 2 (Bob) 0.03, slots 6 and 7 (Bobby)
	// 0.07 and 0.08, so PR = 500·(30+70+80)/1000/3 = 30.
	if got := badges[tid].PR; got < 29.99 || got > 30.01 {
		t.Errorf("Main player Bob: PR = %v, want 30 (both spellings pooled)", got)
	}
}
