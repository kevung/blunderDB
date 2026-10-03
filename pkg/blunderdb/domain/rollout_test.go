package domain

import (
	"math"
	"testing"
	"time"
)

func TestMergeRollouts_OnePerSignature(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a := RolloutAnalysis{Signature: "A", Games: 216, Date: t0}
	aShort := RolloutAnalysis{Signature: "A", Games: 108, Date: t0.Add(time.Hour)}
	b := RolloutAnalysis{Signature: "B", Games: 1296, Date: t0.Add(2 * time.Hour)}

	got := MergeRollouts([]RolloutAnalysis{a}, []RolloutAnalysis{aShort, b})
	if len(got) != 2 || got[0].Signature != "B" || got[1].Games != 216 {
		t.Errorf("merge: %+v — want B first (newest), A keeping its longer series", got)
	}
	if MergeRollouts(nil, nil) != nil {
		t.Error("nothing merged into something")
	}
}

// A position holding only a rollout feeds the columns through it; one with
// another analysis never does (ADR-0060).
func TestColumnSource(t *testing.T) {
	cube := RolloutAnalysis{Signature: "C", Kind: RolloutKindCube, Games: 216, BestCubeAction: "Double, Take",
		Candidates: []RolloutCandidate{{Move: "No double", Equity: 0.4, PlayerWinChance: 70}, {Move: "Double/Take", Equity: 0.6}, {Move: "Double/Pass", Equity: 1}}}
	alone := &PositionAnalysis{Rollouts: []RolloutAnalysis{cube}}
	src := alone.ColumnSource()
	if d := src.DoublingCubeAnalysis; d == nil || d.BestCubeAction != "Double, Take" || math.Abs(d.CubefulNoDoubleError+0.2) > 1e-9 || d.PlayerWinChances != 70 {
		t.Errorf("cube view: %+v", src.DoublingCubeAnalysis)
	}
	if alone.DoublingCubeAnalysis != nil {
		t.Error("ColumnSource wrote into the stored analysis")
	}

	imported := &PositionAnalysis{DoublingCubeAnalysis: &DoublingCubeAnalysis{BestCubeAction: "No Double"}, Rollouts: []RolloutAnalysis{cube}}
	if imported.ColumnSource() != imported {
		t.Error("a rollout moved the columns of an analysed position")
	}

	moves := RolloutAnalysis{Signature: "M", Kind: RolloutKindMoves, Games: 216,
		Candidates: []RolloutCandidate{{Move: "8/5 6/5", Equity: 0.2}, {Move: "13/10 6/5", Equity: 0.15}}}
	m := (&PositionAnalysis{Rollouts: []RolloutAnalysis{moves}}).ColumnSource().CheckerAnalysis.Moves
	if len(m) != 2 || m[0].EquityError != nil || m[1].EquityError == nil || *m[1].EquityError < 0.0499 {
		t.Errorf("checker view: %+v", m)
	}
}

// An import fills the gap a rollout-only position leaves and keeps the
// rollouts of both sides; it never replaces a primary analysis (ADR-0013).
func TestMergeImportedAnalysis(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ra := RolloutAnalysis{Signature: "A", Kind: RolloutKindMoves, Games: 216, Date: t0}
	rb := RolloutAnalysis{Signature: "B", Kind: RolloutKindMoves, Games: 216, Date: t0.Add(time.Hour)}
	xg := &CheckerAnalysis{Moves: []CheckerMove{{Move: "8/5 6/5", AnalysisEngine: "XG"}}}
	gn := &CheckerAnalysis{Moves: []CheckerMove{{Move: "13/10 6/5", AnalysisEngine: "gammonNet"}}}

	rolloutOnly := &PositionAnalysis{Rollouts: []RolloutAnalysis{ra}}
	rolloutOnly.AttachRollout(rb)
	if rolloutOnly.AnalysisType != "" {
		t.Errorf("AttachRollout set AnalysisType %q: an import would read the position as analysed", rolloutOnly.AnalysisType)
	}
	got, changed := MergeImportedAnalysis(rolloutOnly, &PositionAnalysis{AnalysisType: "CheckerMove", CheckerAnalysis: xg})
	if !changed || got.CheckerAnalysis != xg || got.AnalysisType != "CheckerMove" || len(got.Rollouts) != 2 {
		t.Errorf("rollout-only position: changed=%v %+v — want the imported primary beside both rollouts", changed, got)
	}

	analysed := &PositionAnalysis{AnalysisType: "CheckerMove", CheckerAnalysis: gn}
	got, changed = MergeImportedAnalysis(analysed, &PositionAnalysis{AnalysisType: "CheckerMove", CheckerAnalysis: xg, Rollouts: []RolloutAnalysis{ra}})
	if !changed || got.CheckerAnalysis != gn || len(got.Rollouts) != 1 {
		t.Errorf("analysed position: changed=%v %+v — want its primary kept and the imported rollout added", changed, got)
	}

	withRollout := &PositionAnalysis{AnalysisType: "CheckerMove", CheckerAnalysis: gn, Rollouts: []RolloutAnalysis{ra}}
	if _, changed = MergeImportedAnalysis(withRollout, &PositionAnalysis{AnalysisType: "CheckerMove", CheckerAnalysis: xg, Rollouts: []RolloutAnalysis{ra}}); changed {
		t.Error("nothing new imported, yet reported as a change")
	}
	if got, changed = MergeImportedAnalysis(nil, analysed); !changed || got != analysed {
		t.Error("no existing analysis: the imported one is written as it is")
	}
}
