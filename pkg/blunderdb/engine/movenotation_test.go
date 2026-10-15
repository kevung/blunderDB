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

// prisesEn41 : 4-1, un pion sur 6, des blots adverses sur 5 et sur 1. « 6/5* 5/1* »
// et « 6/2 2/1* » ont la même forme, 6/1, et ne laissent pas le même plateau.
func prisesEn41() domain.Position {
	p := domain.InitializePosition()
	p.Dice = [2]int{4, 1}
	p.Board.Points[5] = domain.Point{Checkers: 1, Color: domain.White}
	p.Board.Points[1] = domain.Point{Checkers: 1, Color: domain.White}
	return p
}

func TestHitPoints(t *testing.T) {
	for _, tc := range []struct{ move, want string }{
		{"6/5* 5/1*", "1,5"},
		{"6/2 2/1*", "1"},
		{"8/5*/4", "5"},
		{"13/7*", "7"},
		{"6/5*(2)", "5"},
		{"bar/22* 13/11", "22"},
		{"6/3 3/1", ""},
	} {
		got := ""
		for i, h := range HitPoints(tc.move) {
			if i > 0 {
				got += ","
			}
			got += h
		}
		if got != tc.want {
			t.Errorf("HitPoints(%q) = %q, want %q", tc.move, got, tc.want)
		}
	}
}

func TestPlayKey_DistingueLesPrises(t *testing.T) {
	if PlayKey("6/5* 5/1*") == PlayKey("6/2 2/1*") {
		t.Error("deux coups qui ne frappent pas les mêmes pions ont la même clé")
	}
	if PlayKey("13/7*") != PlayKey("13/8 8/7*") {
		t.Error("XG et gammonNet écrivent le même coup : clés différentes")
	}
	if PlayKey("8/5*/4") != PlayKey("8/5* 5/4") {
		t.Error("un saut chaîné et ses deux pas : clés différentes")
	}
}

// Le joker dépend de la source : une source qui note ses prises dit « pas de
// prise » par l'absence de « * » ; une source qui n'en note jamais ne dit rien.
func TestSamePlay_JokerSelonLaSource(t *testing.T) {
	for _, tc := range []struct {
		a      string
		aMarks bool
		b      string
		bMarks bool
		want   bool
	}{
		{"6/5* 5/1*", true, "6/2 2/1*", true, false},
		{"6/2/1", true, "6/2 2/1*", true, false},
		{"6/1", true, "6/1", true, true},
		{"6/2/1", false, "6/2 2/1*", true, true},
		{"6/2 2/1*", true, "6/2/1", false, true},
		{"6/2/1", false, "13/7", true, false},
	} {
		if got := SamePlay(tc.a, tc.aMarks, tc.b, tc.bMarks); got != tc.want {
			t.Errorf("SamePlay(%q, %v, %q, %v) = %v, want %v", tc.a, tc.aMarks, tc.b, tc.bMarks, got, tc.want)
		}
	}
}

func TestResolvePlay_PrisesEn41(t *testing.T) {
	pos := prisesEn41()
	legal := domain.LegalMoves(&pos)
	double, err := ResolvePlay(legal, "6/5*/1*", true)
	if err != nil {
		t.Fatalf("6/5*/1* : %v", err)
	}
	single, err := ResolvePlay(legal, "6/2/1*", true)
	if err != nil {
		t.Fatalf("6/2/1* : %v", err)
	}
	if double.Result.Board == single.Result.Board {
		t.Fatal("deux prises et une prise laissent le même plateau")
	}
	if double.Notation != "6/5* 5/1*" || single.Notation != "6/2 2/1*" {
		t.Errorf("résolus en %q et %q", double.Notation, single.Notation)
	}
	if _, err := ResolvePlay(legal, "6/1", false); err == nil {
		t.Error("« 6/1 » sans marque d'une source qui n'en note pas : deviné au lieu de refusé")
	}
	if _, err := ResolvePlay(legal, "6/2/1", true); err == nil {
		t.Error("« 6/2/1 » d'une source qui note ses prises ne frappe pas : aucun coup légal")
	}
}
