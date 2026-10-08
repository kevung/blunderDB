// Contract case for the tournament review (ADR-0081): rounds in tournament
// order with the badges' figures, the usual level read from the year before
// the tournament with its own matches left out, and an unknown tournament
// refused. The table that runs it lives in contract.go.
package storagetest

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testStatsTournamentReview(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	// reviewMatch saves a 7-point match of one game; the player of seat 0
	// plays the slots given, the other one the slot after the last. A slot's
	// error grows with it, so matches over different slots differ.
	reviewMatch := func(p1, p2 string, day time.Time, slots []int) int64 {
		t.Helper()
		m := domain.Match{Player1Name: p1, Player2Name: p2, MatchLength: 7, MatchDate: day}
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
		return id
	}
	day := func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

	tid, err := s.Tournaments().Create(ctx, "", "Open", "2025-06-10", "")
	if err != nil {
		t.Fatalf("Create tournament: %v", err)
	}
	otherTID, err := s.Tournaments().Create(ctx, "", "Club", "2025-02-01", "")
	if err != nil {
		t.Fatalf("Create tournament: %v", err)
	}
	r1 := reviewMatch("Alice", "Bob", day(2025, 6, 10), []int{0, 1})
	r2 := reviewMatch("Carol", "Alice", day(2025, 6, 10), []int{2, 3, 4}) // Alice sits second: one decision
	r3 := reviewMatch("Alice", "Dave", day(2025, 6, 11), []int{5})
	for _, id := range []int64{r1, r2, r3} {
		if err := s.Tournaments().AddMatch(ctx, "", tid, id); err != nil {
			t.Fatalf("AddMatch: %v", err)
		}
	}
	// The usual window: four loose matches and one of another tournament
	// in the year before; one after the tournament and one older are out.
	for i, d := range []time.Time{day(2024, 7, 1), day(2024, 12, 1), day(2025, 1, 15), day(2025, 5, 30)} {
		reviewMatch("Alice", "Eve", d, []int{6 + 2*i, 7 + 2*i})
	}
	club := reviewMatch("Alice", "Eve", day(2025, 2, 1), []int{14, 15, 16})
	if err := s.Tournaments().AddMatch(ctx, "", otherTID, club); err != nil {
		t.Fatalf("AddMatch: %v", err)
	}
	reviewMatch("Alice", "Eve", day(2025, 7, 1), []int{18})
	reviewMatch("Alice", "Eve", day(2024, 1, 1), []int{18})

	review, err := s.Stats().TournamentReview(ctx, "", tid, "")
	if err != nil {
		t.Fatalf("TournamentReview: %v", err)
	}
	if review.Player != "Alice" || review.Name != "Open" || review.Matches != 3 {
		t.Fatalf("the default player is the one in the most matches: %+v", review)
	}
	if len(review.Rounds) != 3 {
		t.Fatalf("got %d rounds, want 3", len(review.Rounds))
	}
	badges, err := s.Stats().MatchBadges(ctx, "", []int64{r1, r2, r3})
	if err != nil {
		t.Fatalf("MatchBadges: %v", err)
	}
	decisions := 0
	for i, want := range []struct {
		id       int64
		opponent string
		seat     int
	}{{r1, "Bob", 0}, {r2, "Carol", 1}, {r3, "Dave", 0}} {
		rr := review.Rounds[i]
		b := badges[want.id]
		pr, mwc7 := b.PR, b.MWC7
		if want.seat == 1 {
			pr, mwc7 = b.PR2, b.MWC7P2
		}
		if rr.Round != i+1 || rr.MatchID != want.id || rr.Opponent != want.opponent {
			t.Errorf("round %d: %+v, want match %d against %s", i+1, rr, want.id, want.opponent)
		}
		if math.Abs(rr.PR-pr) > 1e-12 || math.Abs(rr.MWC7.Loss-mwc7.Loss) > 1e-12 {
			t.Errorf("round %d: PR %v L7 %v, badge PR %v L7 %v", i+1, rr.PR, rr.MWC7.Loss, pr, mwc7.Loss)
		}
		decisions += rr.Decisions
	}
	if decisions != 4 || review.Decisions != 4 {
		t.Errorf("Alice played 4 decisions over the rounds (one in the second), got %d (review %d)", decisions, review.Decisions)
	}
	u := review.Usual
	if u.From != "2024-06-10" || u.To != "2025-06-10" || u.Matches != 5 || !u.Available || u.Decisions != 11 {
		t.Errorf("usual level: %+v, want 5 matches and 11 decisions over 2024-06-10..2025-06-10", u)
	}
	if !u.PRInterval.Available {
		t.Errorf("five matches that disagree give the usual PR an interval: %+v", u.PRInterval)
	}
	if review.PRVersus.Verdict != storage.VerdictInsufficient {
		t.Errorf("six decisions are too few for a verdict: %+v", review.PRVersus)
	}
	if len(review.ByPressure) != 4 || review.ByPressure[3].Key != storage.PressureOther || review.ByPressure[3].Decisions != 4 {
		t.Errorf("every decision is at a 4-away score: %+v", review.ByPressure)
	}
	if len(review.ByRank) != storage.TournamentRankSlices || review.ByRank[0].Decisions != 4 {
		t.Errorf("every decision is among the first thirty of its match: %+v", review.ByRank)
	}
	if review.ByClock[0].Decisions+review.ByClock[1].Decisions != 0 {
		t.Errorf("decisions without a time fall in no clock cell: %+v", review.ByClock)
	}
	if review.Families == nil {
		t.Errorf("families are a list, empty when none holds")
	}

	if _, err := s.Stats().TournamentReview(ctx, "", tid+otherTID+100, ""); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("an unknown tournament is ErrNotFound, got %v", err)
	}
	named, err := s.Stats().TournamentReview(ctx, "", tid, "Bob")
	if err != nil {
		t.Fatalf("TournamentReview Bob: %v", err)
	}
	if named.Matches != 1 || len(named.Rounds) != 1 || named.Rounds[0].Opponent != "Alice" || named.Usual.Available {
		t.Errorf("Bob played one round and has no usual level: %+v", named)
	}
}
