package apkg

import (
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Le verso chiffre le coup joué, pas un autre de même forme qui frappe
// autrement : 4-1, « 6/2 2/1* » n'est pas « 6/5* 5/1* ».
func TestBack_ErreurDuCoupJoueSelonSesPrises(t *testing.T) {
	p := &domain.Position{DecisionType: domain.CheckerAction}
	a := &domain.PositionAnalysis{
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "6/5* 5/1*", Equity: 0.10, AnalysisEngine: "XG"},
			{Move: "6/2 2/1*", Equity: 0.02, AnalysisEngine: "XG"},
		}},
		PlayedMoves: []string{"6/2/1*"},
	}
	_, _, played := back(p, a, "en")
	if !strings.Contains(played, "0.080") {
		t.Errorf("verso : %q, attendu l'erreur 0.080 du coup à une prise", played)
	}
}
