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

// testStatsStudyPlan pins the study plan (ADR-0077) on the recurring-error
// families: a family of enough priced errors enters the plan, its recoverable
// MWC is the sum of what MatchDecisionLosses says each error exceeded the
// reference player by, a family short of evidence stays tentative, and an
// error no rule names is only counted.
func testStatsStudyPlan(t *testing.T, s storage.Storage) {
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
	gammon := map[int64]bool{}
	for slot := 0; slot < 6; slot++ {
		gammon[statsCheckerDecision(t, s, gameID, slot, 1, "24/18 13/11", 0.100+0.020*float64(slot), 8)] = true
	}
	statsCubeDecision(t, s, gameID, 6, 1, "No Double", "Double, Take", 0.40, 0.55, 1.00) // missed
	statsCubeDecision(t, s, gameID, 7, 1, "No Double", "Double, Take", 0.40, 0.60, 1.00) // missed
	statsCheckerDecision(t, s, gameID, 8, 1, "24/14", 0.080, 0)                          // no theme

	plan, err := s.Stats().StudyPlan(ctx, "", storage.StatsFilter{DecisionType: -1, PlayerName: "Alice"})
	if err != nil {
		t.Fatalf("StudyPlan: %v", err)
	}
	if plan.NumDecisions != 9 || plan.MinErrors != storage.StudyPlanMinErrors {
		t.Errorf("NumDecisions %d, MinErrors %d; want 9, %d", plan.NumDecisions, plan.MinErrors, storage.StudyPlanMinErrors)
	}
	if plan.Unthemed != 1 || plan.Unpriced != 0 {
		t.Errorf("Unthemed %d, Unpriced %d; want 1, 0", plan.Unthemed, plan.Unpriced)
	}
	if len(plan.Families) != 1 {
		t.Fatalf("Families = %+v, want the gammon family alone", plan.Families)
	}
	f := plan.Families[0]
	if f.Kind != "checker" || f.Theme != "gammon" || f.Errors != 6 || len(f.Positions) != 6 {
		t.Errorf("family = %+v, want six checker gammon errors", f)
	}
	if len(plan.Tentative) != 1 || plan.Tentative[0].Theme != storage.CubeCellOfferMissed || plan.Tentative[0].Errors != 2 {
		t.Errorf("Tentative = %+v, want the two missed doubles", plan.Tentative)
	}

	// The recoverable MWC is the Match panel's own per-decision figures added
	// up: one conversion, one difficulty, whichever view reads them.
	losses, err := s.Stats().MatchDecisionLosses(ctx, "", matchID)
	if err != nil {
		t.Fatalf("MatchDecisionLosses: %v", err)
	}
	var want, sq float64
	byMove := map[int]storage.DecisionLoss{}
	for _, d := range losses {
		byMove[d.MoveNumber] = d
	}
	for slot := 0; slot < 6; slot++ {
		d := byMove[slot]
		if d.MWCLoss == nil || d.Difficulty == nil {
			t.Fatalf("move %d unpriced: %+v", slot, d)
		}
		x := *d.MWCLoss - *d.Difficulty
		want += x
		sq += x * x
	}
	if math.Abs(f.Recoverable-want) > 1e-12 {
		t.Errorf("Recoverable %v, want Σ(ℓ − d) = %v", f.Recoverable, want)
	}
	if half := storage.StudyPlanZ * math.Sqrt(sq); math.Abs(f.Low-(want-half)) > 1e-12 || math.Abs(f.High-(want+half)) > 1e-12 {
		t.Errorf("interval [%v, %v], want %v ± %v", f.Low, f.High, want, half)
	}
	for i, p := range f.Positions {
		if !gammon[p.PositionID] || p.MatchID != matchID || p.Label == "" {
			t.Errorf("position %d = %+v, not a gammon error of match %d", i, p, matchID)
		}
		if i > 0 && p.Excess > f.Positions[i-1].Excess {
			t.Errorf("positions not ranked by excess: %+v", f.Positions)
		}
	}
	queue := plan.QueueEntries(1)
	if len(queue) != 6 || queue[0].Reason != domain.StudyPlan || queue[0].PositionID != f.Positions[0].PositionID {
		t.Errorf("queue = %+v, want the family's six positions, largest excess first", queue)
	}

	bob, err := s.Stats().StudyPlan(ctx, "", storage.StatsFilter{DecisionType: -1, PlayerName: "Bob"})
	if err != nil {
		t.Fatalf("StudyPlan(Bob): %v", err)
	}
	if len(bob.Families)+len(bob.Tentative) != 0 {
		t.Errorf("Bob has Alice's families: %+v", bob)
	}
}
