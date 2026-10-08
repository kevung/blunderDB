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

// testStatsStudyEffect pins the before/after measure (ADR-0079) on both
// backends: a family is studied on the day of the first study mark on one of
// its positions, a match before that day and one after fill the two windows,
// and the change is the difference of their loss rates.
func testStatsStudyEffect(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	if _, err := s.Stats().DateRange(ctx, ""); errors.Is(err, storage.ErrInternal) {
		t.Skip("Stats not implemented on this backend")
	}
	game := func(date time.Time) int64 {
		m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7, MatchDate: date}
		matchID, err := s.Matches().Save(ctx, "", &m)
		if err != nil {
			t.Fatalf("Save match: %v", err)
		}
		gameID, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1})
		if err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
		return gameID
	}
	// The mark is dated now: the first match lies before it, the second after.
	before, after := game(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)), game(time.Now().AddDate(1, 0, 0))
	var studiedID int64
	for slot := 0; slot < 6; slot++ {
		id := statsCheckerDecision(t, s, before, slot, 1, "24/18 13/11", 0.100+0.020*float64(slot), 8)
		if slot == 0 {
			studiedID = id
		}
	}
	for slot := 0; slot < 3; slot++ {
		statsCheckerDecision(t, s, after, slot, 1, "24/18 13/11", 0.100, 8)
	}
	if err := s.ImportBatches().SetStudied(ctx, "", studiedID, true); err != nil {
		t.Fatalf("SetStudied: %v", err)
	}

	got, err := s.Stats().StudyEffect(ctx, "", storage.StatsFilter{DecisionType: -1, PlayerName: "Alice"})
	if err != nil {
		t.Fatalf("StudyEffect: %v", err)
	}
	if len(got.Families) != 1 || got.Unstudied != 0 || got.MinDecisions != storage.StudyEffectMinDecisions {
		t.Fatalf("got %+v, want the gammon family alone, studied", got)
	}
	f := got.Families[0]
	if f.Theme != "gammon" || f.Studied != 1 || f.StudiedOn != time.Now().UTC().Format("2006-01-02") {
		t.Errorf("family %+v, want gammon studied today", f)
	}
	if f.Before.Errors != 6 || f.After.Errors != 3 || f.Before.Decisions != 6 || f.After.Decisions != 3 {
		t.Errorf("windows %+v / %+v, want 6 errors of 6 decisions, then 3 of 3", f.Before, f.After)
	}
	if f.Before.Loss <= 0 || math.Abs(f.Gain-(f.Before.Rate-f.After.Rate)) > 1e-12 || f.Low > f.Gain || f.High < f.Gain {
		t.Errorf("measure %+v", f)
	}
	if f.Verdict != storage.StudyEffectInsufficient {
		t.Errorf("Verdict %q with %d and %d decisions, want insufficient", f.Verdict, f.Before.Decisions, f.After.Decisions)
	}
}

// testStatsDirectionalBiases pins the signed biases (ADR-0079) on both
// backends: takes and passes against the bot's ruling, offers by score, and
// every checker decision with contact read or counted unread.
func testStatsDirectionalBiases(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	if _, err := s.Stats().DateRange(ctx, ""); errors.Is(err, storage.ErrInternal) {
		t.Skip("Stats not implemented on this backend")
	}
	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7,
		MatchDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	gameID, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1})
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	statsCheckerDecision(t, s, gameID, 0, 1, "24/18 13/11", 0.100, 0)
	statsCheckerDecision(t, s, gameID, 1, 1, "24/18 13/11", 0.050, 0)
	statsCubeDecision(t, s, gameID, 2, 1, "Take", "Double, Pass", 0.40, 1.20, 1.00)   // wrong take
	statsCubeDecision(t, s, gameID, 3, 1, "Pass", "Double, Take", 0.40, 0.60, 1.00)   // wrong pass
	statsCubeDecision(t, s, gameID, 4, 1, "Take", "Double, Take", 0.40, 0.60, 1.00)   // right take
	statsCubeDecision(t, s, gameID, 5, 1, "Double", "No Double", 0.40, 0.30, 1.00)    // premature
	statsCubeDecision(t, s, gameID, 6, 1, "Double", "Double, Take", 0.40, 0.60, 1.00) // right double
	statsCubeDecision(t, s, gameID, 7, -1, "Take", "Double, Pass", 0.40, 1.20, 1.00)  // Bob's

	got, err := s.Stats().DirectionalBiases(ctx, "", storage.StatsFilter{DecisionType: -1, PlayerName: "Alice"})
	if err != nil {
		t.Fatalf("DirectionalBiases: %v", err)
	}
	if tp := got.TakePass; tp.Decisions != 3 || tp.Plus != 1 || tp.Minus != 1 || tp.Verdict != storage.BiasInsufficient {
		t.Errorf("TakePass %+v, want 3 decisions, one wrong take, one wrong pass", tp)
	}
	if d := got.Doubles; d.Decisions != 2 || d.Plus != 1 || d.Minus != 0 {
		t.Errorf("Doubles %+v, want 2 offers, one premature", d)
	}
	if len(got.DoublesByScore) != 1 || got.DoublesByScore[0].MoverAway != 4 || got.DoublesByScore[0].Decisions != 2 {
		t.Errorf("DoublesByScore %+v, want the 4-away cell", got.DoublesByScore)
	}
	if got.Blots.Decisions+got.BlotsUnread != 2 {
		t.Errorf("Blots %+v unread %d, want the two checker decisions", got.Blots, got.BlotsUnread)
	}
}
