package storagetest

import (
	"context"
	"testing"

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
