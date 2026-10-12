package storagetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The counts of the biases, the study effect and the time errors open their
// positions through a selection. Each count shown must be the length of the
// list its selection loads: a link that opens more or fewer positions than it
// says is wrong while every figure still looks right.

// assertSelectionLen fails when sel loads other than want positions.
func assertSelectionLen(t *testing.T, s storage.Storage, filter storage.StatsFilter, sel storage.SelectionSpec, want int) {
	t.Helper()
	ids, err := s.Stats().PositionIDsBySelection(context.Background(), "", filter, sel)
	if err != nil {
		t.Fatalf("PositionIDsBySelection(%+v): %v", sel, err)
	}
	if len(ids) != want {
		t.Errorf("PositionIDsBySelection(%+v): %d ids, the count shows %d", sel, len(ids), want)
	}
}

// assertBiasSelections checks every count of b against its "bias" selection.
func assertBiasSelections(t *testing.T, s storage.Storage, filter storage.StatsFilter, b *storage.DirectionalBiases) {
	t.Helper()
	check := func(axis, score string, sb storage.SignedBias) {
		for side, want := range map[string]int{"": sb.Positions, storage.BiasSidePlus: sb.PlusPositions, storage.BiasSideMinus: sb.MinusPositions} {
			assertSelectionLen(t, s, filter, storage.SelectionSpec{Kind: "bias", Bias: axis, BiasScore: score, BiasSide: side}, want)
		}
	}
	check(storage.BiasAxisTakePass, "", b.TakePass)
	check(storage.BiasAxisDoubles, "", b.Doubles)
	check(storage.BiasAxisBlots, "", b.Blots)
	for _, c := range b.DoublesByScore {
		check(storage.BiasAxisDoubles, storage.BiasScoreKey(c), c.SignedBias)
	}
}

// assertStudyEffectSelections checks both windows of every family.
func assertStudyEffectSelections(t *testing.T, s storage.Storage, filter storage.StatsFilter, e *storage.StudyEffect) {
	t.Helper()
	for _, f := range e.Families {
		for window, want := range map[string]int{storage.StudyEffectBefore: f.Before.Positions, storage.StudyEffectAfter: f.After.Positions} {
			assertSelectionLen(t, s, filter, storage.SelectionSpec{Kind: "family_window",
				GameType: f.GameType, FamilyKind: f.Kind, StudiedOn: f.StudiedOn, Window: window}, want)
		}
	}
}

// assertTimeBandSelections checks every row of the time errors; the filter
// passed is one the table does not read, so the selection must ignore it.
func assertTimeBandSelections(t *testing.T, s storage.Storage, rows []storage.TimeErrorRow) {
	t.Helper()
	for _, r := range rows {
		assertSelectionLen(t, s, storage.StatsFilter{DecisionType: -1, PlayerName: "nobody"},
			storage.SelectionSpec{Kind: "time_band", Player: r.Player, TimeBucket: r.Bucket}, r.Positions)
	}
}

// testStatsCountSelectionsUnderFilters holds the count = list rule under the
// filters a selection must apply as its count does: the engine, the period
// and the match length. Two matches, of different lengths and dates, carry
// cube decisions analysed by two engines; checker decisions sit on both sides
// of a study day.
func testStatsCountSelectionsUnderFilters(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	if _, err := s.Stats().DateRange(ctx, ""); errors.Is(err, storage.ErrInternal) {
		t.Skip("Stats not implemented on this backend")
	}
	game := func(length int32, date time.Time) int64 {
		m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: length, MatchDate: date}
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
	cube := func(gameID int64, slot int, played, best, eng string) {
		pos := statsDecisionPos(t, slot)
		pos.DecisionType = domain.CubeAction
		posID, err := s.Positions().Save(ctx, "", &pos)
		if err != nil {
			t.Fatalf("Save cube position: %v", err)
		}
		mv := domain.Move{GameID: gameID, MoveNumber: int32(slot), MoveType: "cube", PositionID: posID, Player: 1, CubeAction: played}
		if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove: %v", err)
		}
		if err := s.Analyses().Save(ctx, "", posID, &domain.PositionAnalysis{
			PlayedCubeActions: []string{played},
			DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{AnalysisEngine: eng, BestCubeAction: best,
				CubefulNoDoubleEquity: 0.40, CubefulDoubleTakeEquity: 0.60, CubefulDoublePassEquity: 1.00,
				CubefulNoDoubleError: -0.100, CubefulDoubleTakeError: -0.100, CubefulDoublePassError: -0.100},
		}); err != nil {
			t.Fatalf("Save cube analysis: %v", err)
		}
	}
	early := game(7, time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC))
	late := game(5, time.Now().AddDate(1, 0, 0))
	cube(early, 0, "Take", "Double, Pass", "XG")
	cube(early, 1, "Pass", "Double, Take", "GNU")
	cube(early, 2, "No Double", "Double, Take", "XG")
	cube(late, 3, "Take", "Double, Pass", "GNU")
	cube(late, 4, "Double", "No Double", "XG")
	var studied int64
	for slot := 5; slot < 8; slot++ {
		id := statsCheckerDecision(t, s, early, slot, 1, "24/18 13/11", 0.100, 8)
		if studied == 0 {
			studied = id
		}
		statsCheckerDecision(t, s, late, slot, 1, "24/18 13/11", 0.100, 8)
	}
	if err := s.ImportBatches().SetStudied(ctx, "", studied, true); err != nil {
		t.Fatalf("SetStudied: %v", err)
	}

	all := storage.StatsFilter{DecisionType: -1, PlayerName: "Alice"}
	filters := map[string]storage.StatsFilter{
		"all":    all,
		"engine": {DecisionType: -1, PlayerName: "Alice", AnalysisEngine: "XG"},
		"period": {DecisionType: -1, PlayerName: "Alice", DateFrom: "2025-01-01", DateTo: "2025-12-31"},
		"length": {DecisionType: -1, PlayerName: "Alice", MatchLength: []int{5}},
	}
	whole, err := s.Stats().DirectionalBiases(ctx, "", all)
	if err != nil {
		t.Fatalf("DirectionalBiases: %v", err)
	}
	for name, f := range filters {
		t.Run(name, func(t *testing.T) {
			b, err := s.Stats().DirectionalBiases(ctx, "", f)
			if err != nil {
				t.Fatalf("DirectionalBiases: %v", err)
			}
			if name != "all" && b.TakePass.Positions+b.Doubles.Positions >= whole.TakePass.Positions+whole.Doubles.Positions {
				t.Errorf("filter %s keeps every cube position (%d), want fewer", name, b.TakePass.Positions+b.Doubles.Positions)
			}
			assertBiasSelections(t, s, f, b)
			e, err := s.Stats().StudyEffect(ctx, "", f)
			if err != nil {
				t.Fatalf("StudyEffect: %v", err)
			}
			if name == "all" && (len(e.Families) == 0 || e.Families[0].Before.Positions == 0 || e.Families[0].After.Positions == 0) {
				t.Errorf("StudyEffect %+v, want a studied family with both windows filled", e)
			}
			assertStudyEffectSelections(t, s, f, e)
		})
	}
}
