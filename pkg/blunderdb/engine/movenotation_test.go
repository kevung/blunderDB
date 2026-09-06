package engine

import "testing"

func TestCanonicalMove_MêmeCoupÉcritAutrement(t *testing.T) {
	for _, tc := range []struct{ a, b, why string }{
		{"13/7", "13/8 8/7", "un pion en deux sauts, ou deux pions : même plateau"},
		{"24/22(2)", "24/23(2) 23/22(2)", "deux pions, chaque saut écrit séparément"},
		{"6/3 3/1", "6/3* 3/1", "la frappe est un fait du plateau, pas du choix"},
		{"4/2(2)", "4/2 4/2*", "la répétition, collapsée d'un côté seulement"},
		{"8/5 6/5", "6/5 8/5", "l'ordre des pas ne dit rien"},
		{"bar/22 13/11", "bar/24 24/22 13/11", "une entrée en deux temps"},
		{"6/off", "6/2 2/off", "une sortie en deux dés"},
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
