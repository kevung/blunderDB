package storage

import (
	"math"
	"testing"
)

func effectRow(id int64, theme, day string, loss float64) StudyPlanRow {
	return StudyPlanRow{RecurringErrorRow: RecurringErrorRow{PositionID: id, GameType: "holding", Kind: "checker",
		Theme: theme}, Day: day, Loss: &loss}
}

// TestBuildStudyEffect_WindowsAndInterval pins ADR-0079: the study day is the
// first study action on a member, the day itself belongs to neither window,
// the change is the before rate minus the after rate with a compound-Poisson
// interval, and an unstudied family is only counted.
func TestBuildStudyEffect_WindowsAndInterval(t *testing.T) {
	var rows []StudyPlanRow
	for i := int64(0); i < 6; i++ {
		rows = append(rows, effectRow(i+1, "blots", "2025-01-10", 0.02))
	}
	rows = append(rows, effectRow(7, "blots", "2025-03-01", 0.01))
	rows = append(rows, effectRow(8, "blots", "2025-02-01", 0.50)) // the study day: neither window
	rows = append(rows, effectRow(9, "gammon", "2025-01-10", 0.03))
	decisions := []StudyEffectDecisions{
		{GameType: "holding", Kind: "checker", Day: "2025-01-10", Count: 40},
		{GameType: "holding", Kind: "checker", Day: "2025-02-01", Count: 7},
		{GameType: "holding", Kind: "checker", Day: "2025-03-01 00:00:00", Count: 50},
		{GameType: "race", Kind: "checker", Day: "2025-03-01", Count: 99},
	}
	studied := map[int64]string{3: "2025-02-15", 2: "2025-02-01"}
	got := BuildStudyEffect(rows, decisions, studied)
	if got.Unstudied != 1 || len(got.Families) != 1 {
		t.Fatalf("got %+v, want one studied family and one unstudied", got)
	}
	f := got.Families[0]
	if f.StudiedOn != "2025-02-01" || f.Studied != 2 {
		t.Errorf("StudiedOn %q Studied %d; want 2025-02-01, 2", f.StudiedOn, f.Studied)
	}
	if f.Before.Decisions != 40 || f.Before.Errors != 6 || f.After.Decisions != 50 || f.After.Errors != 1 {
		t.Errorf("windows %+v / %+v", f.Before, f.After)
	}
	wantGain := 0.12/40 - 0.01/50
	half := StudyPlanZ * math.Sqrt(6*0.02*0.02/1600+0.01*0.01/2500)
	if math.Abs(f.Gain-wantGain) > 1e-12 || math.Abs(f.Low-(wantGain-half)) > 1e-12 {
		t.Errorf("Gain %g Low %g; want %g, %g", f.Gain, f.Low, wantGain, wantGain-half)
	}
	if f.Verdict != StudyEffectImproved {
		t.Errorf("Verdict %q, want improved", f.Verdict)
	}
}

// TestBuildStudyEffect_NoClaimBelowTheMinimum: a window short of decisions
// states no direction, whatever the interval says.
func TestBuildStudyEffect_NoClaimBelowTheMinimum(t *testing.T) {
	rows := []StudyPlanRow{effectRow(1, "blots", "2025-01-10", 0.2), effectRow(2, "blots", "2025-01-11", 0.2)}
	decisions := []StudyEffectDecisions{
		{GameType: "holding", Kind: "checker", Day: "2025-01-10", Count: 100},
		{GameType: "holding", Kind: "checker", Day: "2025-03-01", Count: StudyEffectMinDecisions - 1},
	}
	got := BuildStudyEffect(rows, decisions, map[int64]string{1: "2025-02-01"})
	if f := got.Families[0]; f.Verdict != StudyEffectInsufficient {
		t.Errorf("Verdict %q, want insufficient", f.Verdict)
	}
}
