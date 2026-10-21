package storagetest

import (
	"context"
	"math"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testStatsBadgesFollowWrites: the match and tournament badges, which a
// writer reads from match_stats, follow every write that changes what they
// summarise — a re-analysis, a seat swap, a deletion — and the match badge
// always equals MatchDetail, which reads the decisions.
func testStatsBadgesFollowWrites(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	tid, err := s.Tournaments().Create(ctx, "", "Open", "2025-06-10", "")
	if err != nil {
		t.Fatalf("Create tournament: %v", err)
	}
	matchID, pos := statsFixtureMatch(t, s, 0, "Alice", "Bob")
	if err := s.Tournaments().AddMatch(ctx, "", tid, matchID); err != nil {
		t.Fatalf("AddMatch: %v", err)
	}
	if err := s.LibrarySettings().SetMainPlayer(ctx, "", "Alice"); err != nil {
		t.Fatalf("SetMainPlayer: %v", err)
	}

	near := func(a, b float64) bool { return math.Abs(a-b) <= 1e-12*math.Max(1, math.Abs(a)) }
	check := func(step string, aliceSeat int) storage.MatchBadge {
		t.Helper()
		badges, err := s.Stats().MatchBadges(ctx, "", []int64{matchID})
		if err != nil {
			t.Fatalf("%s: MatchBadges: %v", step, err)
		}
		detail, err := s.Stats().MatchDetail(ctx, "", matchID)
		if err != nil {
			t.Fatalf("%s: MatchDetail: %v", step, err)
		}
		b := badges[matchID]
		if b.PR != detail.Player1.PR || b.PR2 != detail.Player2.PR ||
			!near(b.MWCLoss, detail.Player1.MWCLoss) || !near(b.MWCLoss2, detail.Player2.MWCLoss) ||
			!near(b.MWC7.Loss, detail.Player1.MWC7.Loss) || !near(b.MWC7P2.Loss, detail.Player2.MWC7.Loss) {
			t.Errorf("%s: badge %+v, detail PR %v/%v MWC %v/%v", step, b,
				detail.Player1.PR, detail.Player2.PR, detail.Player1.MWCLoss, detail.Player2.MWCLoss)
		}
		tb, err := s.Stats().TournamentBadges(ctx, "")
		if err != nil {
			t.Fatalf("%s: TournamentBadges: %v", step, err)
		}
		alice := b.PR
		if aliceSeat == 2 {
			alice = b.PR2
		}
		if tb[tid].RefPlayer != "Alice" || tb[tid].PR != alice {
			t.Errorf("%s: tournament badge %+v, Alice's match PR %v", step, tb[tid], alice)
		}
		return b
	}
	before := check("written", 1)

	// Alice's decision re-analysed: her played move now costs four times more.
	eqErr := 0.20
	a := domain.PositionAnalysis{PlayedMoves: []string{"13/11 24/23"},
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "8/6 6/4", Equity: 0.50}, {Move: "13/11 24/23", Equity: 0.30, EquityError: &eqErr}}}}
	if err := s.Analyses().Save(ctx, "", pos[0], &a); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}
	reanalysed := check("re-analysed", 1)
	if reanalysed.PR <= before.PR || reanalysed.PR2 != before.PR2 {
		t.Errorf("re-analysis: PR %v/%v, was %v/%v; want Alice's alone to rise", reanalysed.PR, reanalysed.PR2, before.PR, before.PR2)
	}

	if err := s.Matches().SwapPlayers(ctx, "", matchID); err != nil {
		t.Fatalf("SwapPlayers: %v", err)
	}
	swapped := check("swapped", 2)
	if swapped.PR != reanalysed.PR2 || swapped.PR2 != reanalysed.PR {
		t.Errorf("swap: PR %v/%v, want %v/%v", swapped.PR, swapped.PR2, reanalysed.PR2, reanalysed.PR)
	}

	if err := s.Matches().DeleteCascade(ctx, "", matchID); err != nil {
		t.Fatalf("DeleteCascade: %v", err)
	}
	badges, err := s.Stats().MatchBadges(ctx, "", nil)
	if err != nil {
		t.Fatalf("MatchBadges after delete: %v", err)
	}
	tb, err := s.Stats().TournamentBadges(ctx, "")
	if err != nil {
		t.Fatalf("TournamentBadges after delete: %v", err)
	}
	if _, ok := badges[matchID]; ok {
		t.Errorf("a deleted match keeps its badge")
	}
	if _, ok := tb[tid]; ok {
		t.Errorf("a tournament without its match keeps its badge: %+v", tb[tid])
	}
}
