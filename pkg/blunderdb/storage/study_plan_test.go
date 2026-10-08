package storage

import (
	"math"
	"testing"
)

func planRow(id int64, theme string, loss, diff float64) StudyPlanRow {
	return StudyPlanRow{RecurringErrorRow: RecurringErrorRow{PositionID: id, GameType: "contact", Kind: "checker", Theme: theme, ErrorMP: 100},
		Loss: &loss, Difficulty: &diff}
}

// TestBuildStudyPlan pins ADR-0077's evidence rule and ranking: a family needs
// StudyPlanMinErrors priced errors and a positive lower bound; the plan ranks
// by that bound, so a large but noisy family can fall behind a steady one.
func TestBuildStudyPlan(t *testing.T) {
	var rows []StudyPlanRow
	// steady: ten errors of excess 0.01 each — R 0.10, half-width 1.96·√0.001.
	for i := range 10 {
		rows = append(rows, planRow(int64(100+i), "blots", 0.012, 0.002))
	}
	// noisy: five errors, one larger — a larger R but a lower bound below steady's.
	for i, x := range []float64{0.05, 0.03, 0.03, 0.03, 0.03} {
		rows = append(rows, planRow(int64(200+i), "gammon", x, 0))
	}
	// hard: five errors the reference player makes as often — R ≈ 0, tentative.
	for i := range 5 {
		rows = append(rows, planRow(int64(300+i), "point", 0.01, 0.01))
	}
	// thin: four avoidable errors — below the member floor.
	for i := range 4 {
		rows = append(rows, planRow(int64(400+i), "passive", 0.05, 0))
	}
	rows = append(rows, planRow(500, RecurringThemeNone, 0.05, 0))
	rows = append(rows, StudyPlanRow{RecurringErrorRow: RecurringErrorRow{PositionID: 600, Theme: "blots", Kind: "checker"}})

	p := BuildStudyPlan(rows, 1000, 80)
	if p.Unthemed != 1 || p.Unpriced != 1 {
		t.Errorf("Unthemed %d, Unpriced %d; want 1, 1", p.Unthemed, p.Unpriced)
	}
	var themes []string
	for _, f := range p.Families {
		themes = append(themes, f.Theme)
	}
	if len(themes) != 2 || themes[0] != "blots" || themes[1] != "gammon" {
		t.Fatalf("plan = %v, want [blots gammon] (ranked by lower bound)", themes)
	}
	if p.Families[1].Recoverable <= p.Families[0].Recoverable {
		t.Errorf("the fixture should give gammon the larger R: %+v", p.Families)
	}
	steady := p.Families[0]
	if math.Abs(steady.Recoverable-0.1) > 1e-12 || math.Abs(steady.Low-(0.1-StudyPlanZ*math.Sqrt(0.001))) > 1e-12 {
		t.Errorf("steady = %+v", steady)
	}
	if len(p.Tentative) != 2 || p.Tentative[0].Theme != "passive" || p.Tentative[1].Theme != "point" {
		t.Errorf("Tentative = %+v, want passive then point (by R)", p.Tentative)
	}
	if ids := p.StudyPositionIDs(2); len(ids) != 5 || ids[0] != 200 {
		t.Errorf("rank 2 = %v, want gammon's five positions, the largest excess first", ids)
	}
	if ids := p.StudyPositionIDs(0); len(ids) != 15 {
		t.Errorf("rank 0 = %d positions, want the plan's 15", len(ids))
	}
	if ids := p.StudyPositionIDs(3); len(ids) != 0 {
		t.Errorf("a rank past the plan yields nothing, got %v", ids)
	}
}
