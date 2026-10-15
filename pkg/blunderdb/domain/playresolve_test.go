package domain

import "testing"

// start41 : la position de départ, 4-1 pour Noir, des blots blancs sur les
// points donnés, le point 1 vidé.
func start41(blots ...int) Position {
	p := InitializePosition()
	p.Dice = [2]int{4, 1}
	p.Board.Points[1] = Point{Color: None}
	for _, b := range blots {
		p.Board.Points[b] = Point{Checkers: 1, Color: White}
	}
	return p
}

func boards(t *testing.T, p Position, notation string, marks bool, want int) []Position {
	t.Helper()
	got := PlayBoards(&p, notation, marks)
	if len(got) != want {
		t.Fatalf("%q (marks=%v) : %d plateaux, attendu %d", notation, marks, len(got), want)
	}
	return got
}

// Les pas écrits sont suivis ; un point sauté laisse les deux chemins.
func TestPlayBoards_PasÉcritsEtPointsSautés(t *testing.T) {
	p := start41(2)
	via2 := boards(t, p, "6/2 2/1", false, 1)[0]
	via5 := boards(t, p, "6/5 5/1", false, 1)[0]
	if SameBoard(via2.Board, via5.Board) {
		t.Fatal("passer par le blot de 2 et par 5 laissent le même plateau")
	}
	if via2.Board.Points[BlackBar].Checkers != 0 || via2.Board.Points[WhiteBar].Checkers != 1 {
		t.Error("passer par 2 n'envoie pas le blot blanc sur la barre")
	}
	boards(t, p, "6/1", false, 2)
	if got := boards(t, p, "6/1", true, 1)[0]; !SameBoard(got.Board, via5.Board) {
		t.Error("« 6/1 » d'un dialecte qui note ses prises ne frappe pas : par 5")
	}
	if got := boards(t, p, "6/2*/1", true, 1)[0]; !SameBoard(got.Board, via2.Board) {
		t.Error("le jeton chaîné « 6/2*/1 » est « 6/2* 2/1 »")
	}
	if got := boards(t, p, "2/1 6/2*", true, 1)[0]; !SameBoard(got.Board, via2.Board) {
		t.Error("l'ordre des parties ne change pas le coup")
	}
	boards(t, p, "6/2 2/1", true, 0)   // passer sur un blot sans frapper
	boards(t, p, "6/5* 5/1", false, 0) // une prise sur un point vide
}

func TestPlayBoards_DoublesBarreEtSortie(t *testing.T) {
	p := InitializePosition()
	p.Dice = [2]int{3, 3}
	a := boards(t, p, "8/5(2) 6/3(2)", true, 1)[0]
	b := boards(t, p, "6/3 8/5 6/3 8/5", true, 1)[0]
	if !SameBoard(a.Board, b.Board) {
		t.Error("« (2) » et la répétition laissent des plateaux différents")
	}
	boards(t, p, "8/5(5)", true, 0) // plus de pas que de dés

	bar := InitializePosition()
	bar.Dice = [2]int{3, 1}
	bar.Board.Points[24].Checkers = 1
	bar.Board.Points[BlackBar] = Point{Checkers: 1, Color: Black}
	boards(t, bar, "bar/22 6/5", true, 1)
	boards(t, bar, "25/22 6/5", true, 1)
	boards(t, bar, "8/5 6/5", true, 0) // la barre d'abord

	var off Position
	off.Dice = [2]int{6, 1}
	off.Board.Points[3] = Point{Checkers: 2, Color: Black}
	off.Board.Points[20] = Point{Checkers: 2, Color: White}
	got := boards(t, off, "3/off 3/2", true, 1)[0]
	if got.Board.Bearoff[Black] != 1 {
		t.Errorf("sortie par excédent : %d pions sortis", got.Board.Bearoff[Black])
	}
}

// Blanc lit la numérotation depuis son côté : son 24 est l'index 1.
func TestPlayBoards_Blanc(t *testing.T) {
	p := InitializePosition()
	p.PlayerOnRoll = White
	p.Dice = [2]int{6, 5}
	got := boards(t, p, "24/13", true, 1)[0]
	if got.Board.Points[1].Checkers != 1 || got.Board.Points[12].Checkers != 6 {
		t.Errorf("24/13 de Blanc : index 1 = %d, index 12 = %d", got.Board.Points[1].Checkers, got.Board.Points[12].Checkers)
	}
}

func TestPlayBoards_Illisibles(t *testing.T) {
	p := start41()
	for _, n := range []string{"", "cannot move", "6", "6/0", "*6/2", "6*/2", "off/2", "6/bar", "6/2(9)", "x/y"} {
		if got := PlayBoards(&p, n, false); got != nil {
			t.Errorf("%q : %d plateaux", n, len(got))
		}
	}
	nodice := InitializePosition()
	nodice.Dice = [2]int{}
	if got := PlayBoards(&nodice, "13/9", false); got != nil {
		t.Error("sans dés, aucun coup")
	}
}

func TestSameBoard_IgnoreLaCouleurDesPointsVides(t *testing.T) {
	a := InitializePosition().Board
	b := a
	b.Points[3] = Point{Checkers: 0, Color: Black}
	if !SameBoard(a, b) {
		t.Error("un point vide « noir » et un point vide sans couleur diffèrent")
	}
	b.Points[3].Checkers = 1
	if SameBoard(a, b) {
		t.Error("un pion de plus passe inaperçu")
	}
}
