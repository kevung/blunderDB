package engine

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// board builds a position from a compact map of point → (checkers, colour).
func board(t *testing.T, black, white map[int]int, off [2]int) *domain.Position {
	t.Helper()
	var p domain.Position
	for pt, n := range black {
		p.Board.Points[pt] = domain.Point{Checkers: n, Color: domain.Black}
	}
	for pt, n := range white {
		if p.Board.Points[pt].Checkers > 0 {
			t.Fatalf("les deux camps occupent le point %d", pt)
		}
		p.Board.Points[pt] = domain.Point{Checkers: n, Color: domain.White}
	}
	p.Board.Bearoff = off
	return &p
}

func TestClassifyGamePhase_PositionInitiale(t *testing.T) {
	p := domain.InitializePosition()
	if got := ClassifyGamePhase(&p); got != domain.PhaseOpening {
		t.Fatalf("position initiale classée %v, attendu %v", got, domain.PhaseOpening)
	}
}

func TestClassifyGamePhase_OuverturePuisMilieu(t *testing.T) {
	// Cinq pions noirs ont quitté leurs points de départ (8/3 pions partis de
	// deux, 13/5 partis de trois) : un de plus que OpeningDisplacementMax.
	p := domain.InitializePosition()
	p.Board.Points[8] = domain.Point{Checkers: 1, Color: domain.Black}
	p.Board.Points[13] = domain.Point{Checkers: 2, Color: domain.Black}
	for _, pt := range []int{5, 4, 11, 10, 9} {
		p.Board.Points[pt] = domain.Point{Checkers: 1, Color: domain.Black}
	}
	if got := ClassifyGamePhase(&p); got != domain.PhaseMiddlegame {
		t.Fatalf("cinq pions déplacés classés %v, attendu %v", got, domain.PhaseMiddlegame)
	}

	// Un pion de moins déplacé : encore l'ouverture.
	p.Board.Points[13] = domain.Point{Checkers: 3, Color: domain.Black}
	p.Board.Points[9] = domain.Point{}
	if got := ClassifyGamePhase(&p); got != domain.PhaseOpening {
		t.Fatalf("quatre pions déplacés classés %v, attendu %v", got, domain.PhaseOpening)
	}
}

func TestClassifyGamePhase_UnPionSurLaBarreSortDeLOuverture(t *testing.T) {
	p := domain.InitializePosition()
	p.Board.Points[24] = domain.Point{Checkers: 1, Color: domain.Black}
	p.Board.Points[domain.BlackBar] = domain.Point{Checkers: 1, Color: domain.Black}
	if got := ClassifyGamePhase(&p); got != domain.PhaseMiddlegame {
		t.Fatalf("un pion sur la barre classé %v, attendu %v", got, domain.PhaseMiddlegame)
	}
}

func TestClassifyGamePhase_Course(t *testing.T) {
	// Noir (en marche vers 1) est entièrement devant Blanc : plus de contact,
	// mais des pions hors du jan des deux côtés.
	p := board(t,
		map[int]int{2: 5, 3: 5, 9: 5},
		map[int]int{20: 5, 21: 5, 22: 5},
		[2]int{0, 0})
	if !p.MatchesNoContact() {
		t.Fatal("la position de test devrait être sans contact")
	}
	if got := ClassifyGamePhase(p); got != domain.PhaseRace {
		t.Fatalf("course classée %v, attendu %v", got, domain.PhaseRace)
	}
}

func TestClassifyGamePhase_Bearoff(t *testing.T) {
	p := board(t,
		map[int]int{1: 5, 2: 5, 3: 5},
		map[int]int{22: 5, 23: 5, 24: 5},
		[2]int{0, 0})
	if got := ClassifyGamePhase(p); got != domain.PhaseBearoff {
		t.Fatalf("bearoff classé %v, attendu %v", got, domain.PhaseBearoff)
	}

	// Un seul pion blanc hors de son jan suffit à ramener la course.
	p.Board.Points[24] = domain.Point{Checkers: 4, Color: domain.White}
	p.Board.Points[18] = domain.Point{Checkers: 1, Color: domain.White}
	if got := ClassifyGamePhase(p); got != domain.PhaseRace {
		t.Fatalf("un pion hors du jan classé %v, attendu %v", got, domain.PhaseRace)
	}
}

func TestClassifyGamePhase_BearoffAvecPionsSortis(t *testing.T) {
	p := board(t,
		map[int]int{1: 3, 2: 2},
		map[int]int{23: 4, 24: 1},
		[2]int{10, 10})
	if got := ClassifyGamePhase(p); got != domain.PhaseBearoff {
		t.Fatalf("fin de bearoff classée %v, attendu %v", got, domain.PhaseBearoff)
	}
}

func TestClassifyGamePhase_ContactProfondNestPasUneCourse(t *testing.T) {
	// Blanc tient encore le point 1 (l'ace-point adverse) : contact, milieu.
	p := board(t,
		map[int]int{2: 5, 3: 5, 4: 5},
		map[int]int{1: 2, 20: 8, 21: 5},
		[2]int{0, 0})
	if p.MatchesNoContact() {
		t.Fatal("le contact devrait subsister")
	}
	if got := ClassifyGamePhase(p); got != domain.PhaseMiddlegame {
		t.Fatalf("contact profond classé %v, attendu %v", got, domain.PhaseMiddlegame)
	}
}

func TestGamePhaseTokensAllerRetour(t *testing.T) {
	for phase, name := range domain.GamePhaseNames {
		if phase.String() != name {
			t.Errorf("%d.String() = %q, attendu %q", phase, phase.String(), name)
		}
		got, ok := domain.ParseGamePhase(name)
		if !ok || got != phase {
			t.Errorf("ParseGamePhase(%q) = %v, %v ; attendu %v, true", name, got, ok, phase)
		}
	}
	if _, ok := domain.ParseGamePhase("holding"); ok {
		t.Error("« holding » est un type de jeu, pas une phase : il ne doit pas se parser")
	}
}

// ClassifyGamePhase ne lit que le plateau : le camp au trait ne change rien.
func TestClassifyGamePhase_SymetriqueAuTrait(t *testing.T) {
	p := domain.InitializePosition()
	p.PlayerOnRoll = domain.Black
	a := ClassifyGamePhase(&p)
	p.PlayerOnRoll = domain.White
	if b := ClassifyGamePhase(&p); a != b {
		t.Fatalf("la phase dépend du camp au trait : %v puis %v", a, b)
	}
}
