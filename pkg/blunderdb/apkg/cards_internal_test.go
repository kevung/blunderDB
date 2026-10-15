package apkg

import (
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Le verso chiffre le coup joué, pas un autre de même forme qui frappe
// autrement : 4-1, « 6/2 2/1* » n'est pas « 6/5* 5/1* ».
func TestBack_ErreurDuCoupJoueSelonSesPrises(t *testing.T) {
	start := domain.InitializePosition()
	p := &start
	p.DecisionType = domain.CheckerAction
	p.Dice = [2]int{4, 1}
	p.Board.Points[5] = domain.Point{Checkers: 1, Color: domain.White}
	p.Board.Points[1] = domain.Point{Checkers: 1, Color: domain.White}
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

// Un coup joué sans marque, de source inconnue, suit ses pas : « 6/2 2/1 » passe
// par le blot de 2 et y frappe. C'est le « 6/2* 2/1 » de gammonNet, pas son
// « 6/5 5/1 » de même forme.
func TestBack_UnCoupSansMarqueSuitSesPas(t *testing.T) {
	start := domain.InitializePosition()
	p := &start
	p.DecisionType = domain.CheckerAction
	p.Dice = [2]int{4, 1}
	p.Board.Points[1] = domain.Point{Color: domain.None}
	p.Board.Points[2] = domain.Point{Checkers: 1, Color: domain.White}
	a := &domain.PositionAnalysis{
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "6/5 5/1", Equity: 0.10, AnalysisEngine: "gammonNet v1.7.0"},
			{Move: "6/2* 2/1", Equity: 0.03, AnalysisEngine: "gammonNet v1.7.0"},
		}},
		PlayedMoves: []string{"6/2 2/1"},
	}
	_, _, played := back(p, a, "en")
	if !strings.Contains(played, "0.070") {
		t.Errorf("verso : %q, attendu l'erreur 0.070 du coup qui frappe sur 2", played)
	}
}
