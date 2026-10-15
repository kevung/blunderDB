package gammonnet

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// La comparaison porte sur la DÉCISION, pas sur sa notation : deux moteurs qui
// jouent le même coup en le notant différemment sont d'accord.
func TestCompareOne_LeDialecteNEstPasUnDesaccord(t *testing.T) {
	stored := &domain.PositionAnalysis{
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "13/7", Equity: 0.10, AnalysisEngine: "XG"},
			{Move: "24/18", Equity: 0.05, AnalysisEngine: "XG"},
		}},
	}
	// Ce que gammonNet répondrait, écrit dans son dialecte à lui.
	got := compareCheckerAnswer(stored, "13/8 8/7", domain.PhaseOpening.String(), 1)
	if !got.same {
		t.Errorf("« 13/7 » et « 13/8 8/7 » sont le même coup ; comparés comme différents")
	}
	if got.dis.Cost != 0 {
		t.Errorf("un accord coûte %v", got.dis.Cost)
	}
}

func TestCompareOne_UnVraiDesaccordEstChiffre(t *testing.T) {
	stored := &domain.PositionAnalysis{
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "13/7", Equity: 0.10, AnalysisEngine: "XG"},
			{Move: "24/18", Equity: 0.05, AnalysisEngine: "XG"},
		}},
	}
	got := compareCheckerAnswer(stored, "24/18", domain.PhaseOpening.String(), 1)
	if got.same {
		t.Fatal("deux coups différents comptés comme identiques")
	}
	if want := 0.05; got.dis.Cost < want-1e-9 || got.dis.Cost > want+1e-9 {
		t.Errorf("coût du désaccord : %v, attendu %v (0,10 − 0,05)", got.dis.Cost, want)
	}
}

// Un coup que le moteur importé n'a jamais listé ne peut pas être tarifé sur
// son échelle : on ne l'invente pas.
func TestCompareOne_UnCoupNonListeNeSeTarifePas(t *testing.T) {
	stored := &domain.PositionAnalysis{
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "13/7", Equity: 0.10, AnalysisEngine: "XG"},
		}},
	}
	got := compareCheckerAnswer(stored, "24/18", domain.PhaseOpening.String(), 1)
	if got.same {
		t.Fatal("deux coups différents comptés comme identiques")
	}
	if got.dis.Cost != 0 {
		t.Errorf("un coup absent de la liste stockée est tarifé %v ; il ne peut pas l'être", got.dis.Cost)
	}
}

// Les moteurs n'écrivent pas les actions de videau de la même façon. La
// comparaison porte sur la substance : doubler ou non, pris ou passé.
func TestCubeActionKind_LesDialectesSeRejoignent(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		same bool
	}{
		{"No Double", "No double", true},
		{"Double, take", "Double/Take", true},
		{"Double, pass", "Double/Pass", true},
		{"Too good to double, pass", "No double", true},
		{"No double", "Double, take", false},
		{"Double, take", "Double, pass", false},
	} {
		if got := cubeActionKind(tc.a) == cubeActionKind(tc.b); got != tc.same {
			t.Errorf("%q vs %q : même décision = %v, attendu %v", tc.a, tc.b, got, tc.same)
		}
	}
}

func TestAggregate_UnRefusNEstPasUnEchec(t *testing.T) {
	cmp := Aggregate([]ComparisonSample{
		{outcome: GNRefused},
		{outcome: GNFailed},
		{same: true, dis: ComparisonDisagreement{Phase: "race"}},
		{same: false, dis: ComparisonDisagreement{Phase: "race", Cost: 0.20}},
	})
	if cmp.Refused != 1 || cmp.Failed != 1 {
		t.Errorf("refusés %d, échecs %d ; attendu 1 et 1", cmp.Refused, cmp.Failed)
	}
	if cmp.Compared != 2 || cmp.SameBest != 1 {
		t.Errorf("comparés %d dont %d d'accord ; attendu 2 et 1", cmp.Compared, cmp.SameBest)
	}
	if cmp.OverThreshold != 1 {
		t.Errorf("au-dessus du seuil : %d, attendu 1", cmp.OverThreshold)
	}
	if b := cmp.ByPhase["race"]; b.Compared != 2 || b.SameBest != 1 {
		t.Errorf("ventilation par phase : %+v", b)
	}
	if len(cmp.Worst) != 1 || cmp.Worst[0].Cost != 0.20 {
		t.Errorf("pires désaccords : %+v", cmp.Worst)
	}
}

// Deux coups de même forme qui ne frappent pas les mêmes pions sont deux
// décisions : 4-1, « 6/5* 5/1* » contre « 6/2 2/1* ».
func TestCompareOne_LesPrisesDistinguentDeuxCoups(t *testing.T) {
	stored := &domain.PositionAnalysis{
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "6/5* 5/1*", Equity: 0.10, AnalysisEngine: "XG"},
			{Move: "6/2 2/1*", Equity: 0.02, AnalysisEngine: "XG"},
		}},
	}
	got := compareCheckerAnswer(stored, "6/2 2/1*", domain.PhaseMiddlegame.String(), 1)
	if got.same {
		t.Fatal("une prise et deux prises comptées comme le même coup")
	}
	if want := 0.08; got.dis.Cost < want-1e-9 || got.dis.Cost > want+1e-9 {
		t.Errorf("coût du désaccord : %v, attendu %v", got.dis.Cost, want)
	}
}

// XG écrit ses prises même quand sa liste n'en contient aucune : son
// « 6/2/1 » ne frappe pas, le « 6/5*/1 » de gammonNet frappe.
func TestCompareOne_LeDialecteDitLesPrisesMemeSansEtoile(t *testing.T) {
	stored := &domain.PositionAnalysis{
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "6/2/1", Equity: 0.10, AnalysisEngine: "XG"},
			{Move: "13/9 6/5", Equity: 0.04, AnalysisEngine: "XG"},
		}},
	}
	got := compareCheckerAnswer(stored, "6/5*/1", domain.PhaseMiddlegame.String(), 1)
	if got.same {
		t.Fatal("6/5*/1 et 6/2/1 comptés comme le même coup")
	}
	if got.dis.Cost != 0 {
		t.Errorf("coût %v : le coup de gammonNet n'est pas listé, il ne se tarifie pas", got.dis.Cost)
	}
}
