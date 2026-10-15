package engine

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

func TestCanonicalMove_MêmeCoupÉcritAutrement(t *testing.T) {
	for _, tc := range []struct{ a, b, why string }{
		{"13/7", "13/8 8/7", "un pion en deux sauts, ou deux pions : même plateau"},
		{"24/22(2)", "24/23(2) 23/22(2)", "deux pions, chaque saut écrit séparément"},
		{"6/3 3/1", "6/3* 3/1", "la forme ne porte pas la frappe : SamePlay la lit"},
		{"4/2(2)", "4/2 4/2*", "la répétition, collapsée d'un côté seulement"},
		{"8/5 6/5", "6/5 8/5", "l'ordre des pas ne dit rien"},
		{"bar/22 13/11", "bar/24 24/22 13/11", "une entrée en deux temps"},
		{"6/off", "6/2 2/off", "une sortie en deux dés"},
		{"8/5*/4", "8/5* 5/4", "un pion en deux sauts écrit d'un seul jeton"},
		{"13/10/7(2)", "13/7(2)", "le jeton chaîné répété"},
	} {
		if got, want := CanonicalMove(tc.a), CanonicalMove(tc.b); got != want {
			t.Errorf("%s :\n  %q → %q\n  %q → %q", tc.why, tc.a, got, tc.b, want)
		}
	}
}

func TestCanonicalMove_CoupsDifférentsRestentDifférents(t *testing.T) {
	for _, tc := range []struct{ a, b string }{
		{"13/7", "13/8 6/5"},
		{"8/5 6/5", "8/5 8/7"},
		{"24/22(2)", "24/22"},
		{"6/off", "5/off"},
	} {
		if CanonicalMove(tc.a) == CanonicalMove(tc.b) {
			t.Errorf("%q et %q sont deux coups différents et se confondent en %q",
				tc.a, tc.b, CanonicalMove(tc.a))
		}
	}
}

// Un non-coup ("cannot move") ne doit pas faire paniquer la fonction ni se
// confondre avec un coup.
func TestCanonicalMove_NonCoup(t *testing.T) {
	if got := CanonicalMove(""); got != "" {
		t.Errorf("le coup vide donne %q", got)
	}
	if CanonicalMove("cannot move") == CanonicalMove("13/7") {
		t.Error("« cannot move » se confond avec un coup")
	}
}

// position41 : 4-1 pour Noir, la position de départ avec des blots blancs sur
// les points donnés (le point 1 vidé sauf s'il est nommé).
func position41(blots ...int) domain.Position {
	p := domain.InitializePosition()
	p.Dice = [2]int{4, 1}
	p.Board.Points[1] = domain.Point{Color: domain.None}
	for _, b := range blots {
		p.Board.Points[b] = domain.Point{Checkers: 1, Color: domain.White}
	}
	return p
}

// prisesEn41 : 4-1, des blots blancs sur 5 et sur 1. « 6/5* 5/1* » et
// « 6/2 2/1* » ont la même forme, 6/1, et ne laissent pas le même plateau.
func prisesEn41() domain.Position { return position41(5, 1) }

func resolve(t *testing.T, pos domain.Position, notation string, marks bool) domain.Board {
	t.Helper()
	b, err := ResolveBoard(&pos, notation, marks)
	if err != nil {
		t.Fatalf("%q : %v", notation, err)
	}
	return b.Board
}

func TestResolveBoard_PrisesEn41(t *testing.T) {
	pos := prisesEn41()
	double := resolve(t, pos, "6/5*/1*", true)
	single := resolve(t, pos, "6/2/1*", true)
	if double == single {
		t.Fatal("deux prises et une prise laissent le même plateau")
	}
	if resolve(t, pos, "6/5* 5/1*", true) != double || resolve(t, pos, "6/2 2/1*", true) != single {
		t.Error("le même coup écrit pas à pas ne laisse pas le même plateau")
	}
	if resolve(t, pos, "2/1* 6/2", true) != single {
		t.Error("l'ordre des parties change le plateau")
	}
	if _, err := ResolveBoard(&pos, "6/1", false); err == nil {
		t.Error("« 6/1 » sans marque d'une source qui n'en note pas : deviné au lieu de refusé")
	}
	if _, err := ResolveBoard(&pos, "6/2/1", true); err == nil {
		t.Error("« 6/2/1 » d'une source qui note ses prises ne frappe pas : aucun coup")
	}
	if resolve(t, pos, "6/2/1", false) != single {
		t.Error("« 6/2/1 » lu avec tolérance passe par 2 et frappe sur 1")
	}
}

// Le dialecte d'un moteur qui note ses prises dit aussi les prises absentes ;
// celui qui ne les note pas laisse la prise possible, mais suit les pas écrits.
func TestResolveBoard_PasÉcritsEtPrisesPossibles(t *testing.T) {
	pos := position41(2)
	via5 := resolve(t, pos, "6/5 5/1", true)
	via2 := resolve(t, pos, "6/2* 2/1", true)
	if via5 == via2 {
		t.Fatal("passer par 2 en frappant et par 5 laissent le même plateau")
	}
	if resolve(t, pos, "6/2 2/1", false) != via2 {
		t.Error("« 6/2 2/1 » d'une source sans marques : passe par 2, donc frappe")
	}
	if _, err := ResolveBoard(&pos, "6/2 2/1", true); err == nil {
		t.Error("« 6/2 2/1 » d'une source qui note ses prises ne peut pas passer par le blot sans frapper")
	}
	if _, err := ResolveBoard(&pos, "6/5* 5/1", false); err == nil {
		t.Error("« 6/5* 5/1 » frappe sur 5 où il n'y a rien : aucun coup")
	}
	if _, err := ResolveBoard(&pos, "6/1", false); err == nil {
		t.Error("« 6/1 » sans marque : par 5 ou par 2 en frappant, deviné au lieu de refusé")
	}
	if resolve(t, pos, "6/1", true) != via5 {
		t.Error("« 6/1 » d'une source qui note ses prises ne frappe pas : par 5")
	}
}

func TestResolveBoard_NotationsIllisibles(t *testing.T) {
	pos := position41()
	for _, n := range []string{"", "cannot move", "6/0", "*6/2", "bar/21", "6/2 6/2 6/2"} {
		if _, err := ResolveBoard(&pos, n, false); err == nil {
			t.Errorf("%q résolu", n)
		}
	}
	nodice := domain.InitializePosition()
	if _, err := ResolveBoard(&nodice, "13/9 6/5", false); err == nil {
		t.Error("une position sans dés résout un coup")
	}
}

func TestEngineMarksHits_RolloutStockéLuAvecTolérance(t *testing.T) {
	if marks, known := EngineMarksHits("blunderDB rollout v1 / gammonNet v1.7.0"); !known || marks {
		t.Errorf("un rollout stocké nomme ses coups comme on les lui a donnés : marks=%v known=%v", marks, known)
	}
	if marks, _ := EngineMarksHits("gammonNet v1.7.0"); !marks {
		t.Error("gammonNet note ses prises")
	}
}

func TestCandidateMarksHits_ParDialecte(t *testing.T) {
	marks := CandidateMarksHits([]domain.CheckerMove{
		{Move: "6/2/1", AnalysisEngine: "XG"},
		{Move: "6/3* 3/1", AnalysisEngine: "GNUbg"},
		{Move: "13/8 8/7", AnalysisEngine: "gammonNet v1.7.0"},
		{Move: "24/18", AnalysisEngine: "Inconnu"},
		{Move: "6/5*", AnalysisEngine: "Autre"},
	})
	for engine, want := range map[string]bool{"XG": true, "GNUbg": false, "gammonNet v1.7.0": true, "Inconnu": false, "Autre": true} {
		if marks[engine] != want {
			t.Errorf("%s : marque ses prises = %v, attendu %v", engine, marks[engine], want)
		}
	}
}
