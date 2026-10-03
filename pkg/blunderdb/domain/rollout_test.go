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
